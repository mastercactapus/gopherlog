package logstash

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Parser struct {
	s    *Scanner
	line int
	col  int
	buf  struct {
		tok  Token
		lit  string
		n    int
		line int
		col  int
	}
}

func NewParser(r io.Reader) *Parser {
	return &Parser{s: NewScanner(r)}
}

func (p *Parser) scan() (tok Token, lit string) {
	if p.buf.n != 0 {
		p.buf.n = 0
		return p.buf.tok, p.buf.lit
	}
	p.buf.line, p.buf.col = p.s.Pos()
	tok, lit = p.s.Scan()
	p.line, p.col = p.s.Pos()
	p.buf.tok = tok
	p.buf.lit = lit
	return tok, lit
}
func (p *Parser) unscan() { p.buf.n = 1 }
func (p *Parser) Pos() (line, col int) {
	if p.buf.n != 0 {
		return p.buf.line, p.buf.col
	}
	return p.line, p.col
}
func (p *Parser) PrevPos() (line, col int) {
	return p.buf.line, p.buf.col
}

func (p *Parser) scanIgnoreWhitespaceComment() (tok Token, lit string) {
	tok, lit = p.scan()
	for tok == TokenWhitespace || tok == TokenComment {
		tok, lit = p.scan()
	}
	return tok, lit
}

func (p *Parser) getToken(types ...Token) (tok Token, lit string, err error) {
	tok, lit = p.scanIgnoreWhitespaceComment()
	if tok == TokenIllegal {
		return tok, lit, p.errIllegal(lit)
	}
	for _, t := range types {
		if tok == t {
			return tok, lit, nil
		}
	}
	l, c := p.PrevPos()
	if len(types) == 1 {
		return tok, lit, fmt.Errorf("unexpected token '%s' expected '%s' on line %d col %d", lit, types[0].String(), l, c)
	}

	list := make([]string, len(types))
	for i, t := range types {
		list[i] = t.String()
	}

	return tok, lit, fmt.Errorf("unexpected token '%s' expected one of %s on line %d col %d", lit, strings.Join(list, "|"), l, c)
}

func (p *Parser) errUnexpected(expected, lit string) error {
	l, c := p.PrevPos()
	return fmt.Errorf("unexpected token '%s' expected '%s' on line %d col %d", lit, expected, l, c)
}
func (p *Parser) errIllegal(lit string) error {
	l, c := p.PrevPos()
	return fmt.Errorf("illegal token '%s' on line %d col %d", lit, l, c)
}
func (p *Parser) err(err error) error {
	l, c := p.PrevPos()
	return fmt.Errorf("parse error: %s at line %d col %d", err.Error(), l, c)
}

func (p *Parser) parseArray() ([]string, error) {
	tok, lit, err := p.getToken(TokenLBracket)
	if err != nil {
		return nil, err
	}
	vals := make([]string, 0, 20)
	for {
		tok, lit, err = p.getToken(TokenRBracket, TokenString, TokenComma)
		if err != nil {
			return nil, err
		}
		if tok == TokenComma {
			continue
		}
		if tok == TokenRBracket {
			break
		}
		str, err := strconv.Unquote(lit)
		if err != nil {
			return nil, p.err(err)
		}
		vals = append(vals, str)
	}
	return vals, nil
}

func (p *Parser) parseField() (interface{}, error) {
	tok, lit, err := p.getToken(TokenAssignment)
	if err != nil {
		return nil, err
	}
	tok, lit, err = p.getToken(TokenString, TokenNumber, TokenLBracket)
	if err != nil {
		return nil, err
	}

	switch tok {
	case TokenString:
		return strconv.Unquote(lit)
	case TokenLBracket:
		p.unscan()
		return p.parseArray()
	case TokenNumber:
		if strings.Contains(lit, ".") {
			return strconv.ParseFloat(lit, 64)
		}
		return strconv.ParseInt(lit, 10, 64)
	}
	panic("unexpected token: " + tok.String())
}

func (p *Parser) parsePluginFields() (PluginFields, error) {
	tok, lit, err := p.getToken(TokenLCurlyBrace)
	if err != nil {
		return nil, err
	}

	fields := make(PluginFields, 20)
	for {
		tok, lit, err = p.getToken(TokenIdentifier, TokenRCurlyBrace)
		if err != nil {
			return nil, err
		}
		if tok == TokenRCurlyBrace {
			break
		}
		field, err := p.parseField()
		if err != nil {
			return nil, err
		}
		if fields[lit] != nil {
			strs, ok := fields[lit].([]string)
			if !ok {
				return nil, p.err(fmt.Errorf("field '%s' already delcared and not an array", lit))
			}
			str, ok := field.(string)
			if !ok {
				strs2, ok := field.([]string)
				if !ok {
					return nil, p.err(fmt.Errorf("field '%s' is an array but new value is not a string or []string", lit))
				}
				field = append(strs, strs2...)
			} else {

				field = append(strs, str)
			}
		}
		fields[lit] = field
	}

	return fields, nil
}

func (p *Parser) parseSectionBlock() ([]PluginConfig, error) {
	tok, lit, err := p.getToken(TokenLCurlyBrace)
	if err != nil {
		return nil, err
	}

	cfgs := make([]PluginConfig, 0, 20)
	for {
		tok, lit, err = p.getToken(TokenIdentifier, TokenRCurlyBrace)
		if err != nil {
			return nil, err
		}
		if tok == TokenRCurlyBrace {
			break
		}

		fields, err := p.parsePluginFields()
		if err != nil {
			return nil, err
		}
		cfgs = append(cfgs, PluginConfig{
			Name:   lit,
			Fields: fields,
		})
	}

	return cfgs, nil
}

func (p *Parser) Parse() (*Config, error) {
	cfg := &Config{
		Input:  make([]PluginConfig, 0, 20),
		Filter: make([]PluginConfig, 0, 20),
		Output: make([]PluginConfig, 0, 20),
	}
	var err error
	var lit string
	var tok Token
	for {
		tok, lit, err = p.getToken(TokenIdentifier)
		if tok == TokenEOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if lit != "input" && lit != "output" && lit != "filter" {
			return nil, p.errUnexpected("input|output|filter", lit)
		}
		cfgs, err := p.parseSectionBlock()
		if err != nil {
			return nil, err
		}
		switch lit {
		case "input":
			cfg.Input = append(cfg.Input, cfgs...)
		case "filter":
			cfg.Filter = append(cfg.Filter, cfgs...)
		case "output":
			cfg.Output = append(cfg.Output, cfgs...)
		}
	}
	return cfg, nil
}
