package logstash

import (
	"bufio"
	"bytes"
	"io"
)

var eof = rune(0)

type positionTracker struct {
	io.RuneScanner
	line     int
	col      int
	lastLine int
	lastCol  int
}

func (p *positionTracker) ReadRune() (r rune, size int, err error) {
	r, size, err = p.RuneScanner.ReadRune()
	p.lastLine = p.line
	p.lastCol = p.col
	if r == '\n' {
		p.line++
		p.col = 1
	} else {
		p.col++
	}
	return r, size, err
}
func (p *positionTracker) UnreadRune() error {
	err := p.RuneScanner.UnreadRune()
	if err == nil {
		p.line = p.lastLine
		p.col = p.lastCol
	}
	return err

}
func (p *positionTracker) Pos() (line, col int) {
	return p.line, p.col
}

func NewpositionTracker(r io.RuneScanner) *positionTracker {
	return &positionTracker{RuneScanner: r, line: 1, lastLine: 1}
}

type Scanner struct {
	r *positionTracker
}

func (s *Scanner) Pos() (line int, col int) {
	return s.r.Pos()
}
func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}
func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}
func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

// operators could be the start of assignment or comparators
// =, >, <, or !
func isOperator(ch rune) bool {
	return (ch >= '<' && ch <= '>') || ch == '!'
}

func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: NewpositionTracker(bufio.NewReader(r))}
}

func (s *Scanner) scanOperator() (tok Token, lit string) {
	ch := s.read()
	ch2 := s.read()
	switch ch {
	case '=':
		switch ch2 {
		case '~':
			return TokenEqRegex, "=~"
		case '=':
			return TokenEq, "=="
		case '>':
			return TokenAssignment, "=>"
		default:
			return TokenIllegal, string(ch) + string(ch2)
		}
	case '!':
		switch ch2 {
		case '=':
			return TokenNEq, "!="
		case '~':
			return TokenNEqRegex, "!~"
		default:
			s.unread()
			return TokenNegate, "!"
		}
	case '>':
		switch ch2 {
		case '=':
			return TokenGTE, ">="
		default:
			s.unread()
			return TokenGT, ">"
		}
	case '<':
		switch ch2 {
		case '=':
			return TokenLTE, ">="
		default:
			s.unread()
			return TokenLT, ">"
		}
	}
	s.unread()
	return TokenIllegal, string(ch)
}

// scanWhitespace will scan subsequent whitespace exiting on EOF or non-whitespace
func (s *Scanner) scanWhitespace() (tok Token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())
	var ch rune
	for {
		if ch = s.read(); ch == eof {
			break
		} else if !isWhitespace(ch) {
			s.unread()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}
	return TokenWhitespace, buf.String()
}

func (s *Scanner) scanRegex() (tok Token, lit string) {
	return s.scanSequence(TokenRegex, '/', false)
}

func (s *Scanner) scanString() (tok Token, lit string) {
	tok, lit = s.scanSequence(TokenString, '"', true)
	lit = "\"" + lit + "\""
	return tok, lit
}

func (s *Scanner) scanSequence(tokenType Token, delim rune, newline bool) (tok Token, lit string) {
	var buf bytes.Buffer
	s.read()
	var ch rune
	var escaped bool
	for {
		if ch = s.read(); ch == eof {
			return TokenIllegal, buf.String()
		} else if !newline && ch == '\n' {
			return TokenIllegal, buf.String()
		} else if !escaped && ch == delim {
			break
		} else {
			buf.WriteRune(ch)
		}
		if !escaped && ch == '\\' {
			escaped = true
		} else {
			escaped = false
		}
	}
	return tokenType, buf.String()
}

func (s *Scanner) scanNumber() (tok Token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())
	var ch rune
	var hasDecimal bool
	for {
		if ch = s.read(); ch == eof {
			break
		} else if hasDecimal && ch == '.' {
			return TokenIllegal, buf.String()
		} else if isLetter(ch) {
			return TokenIllegal, buf.String()
		} else if !isDigit(ch) {
			s.unread()
			break
		}
		if ch == '.' {
			hasDecimal = true
		}
		_, _ = buf.WriteRune(ch)
	}
	return TokenNumber, buf.String()
}

func (s *Scanner) scanComment() (tok Token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())
	var ch rune
	for {
		if ch = s.read(); ch == eof {
			break
		} else if ch == '\n' {
			s.unread()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}
	return TokenComment, buf.String()
}

func (s *Scanner) scanIdentifier() (tok Token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())
	var ch rune
	for {
		if ch = s.read(); ch == eof {
			break
		} else if !isLetter(ch) && !isDigit(ch) && ch != '_' && ch != '@' && ch != '-' {
			s.unread()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}
	str := buf.String()
	switch str {
	case "and":
		return TokenAnd, str
	case "or":
		return TokenOr, str
	case "nand":
		return TokenNand, str
	case "xor":
		return TokenXor, str
	case "if":
		return TokenIf, str
	case "else":
		return TokenElse, str
	case "not":
		return TokenNot, str
	case "in":
		return TokenIn, str
	}

	return TokenIdentifier, str
}

func (s *Scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}

func (s *Scanner) unread() { _ = s.r.UnreadRune() }

func (s *Scanner) Scan() (tok Token, lit string) {
	ch := s.read()
	if isWhitespace(ch) {
		s.unread()
		return s.scanWhitespace()
	} else if isLetter(ch) || ch == '@' {
		s.unread()
		return s.scanIdentifier()
	} else if isOperator(ch) {
		s.unread()
		return s.scanOperator()
	} else if isDigit(ch) {
		s.unread()
		return s.scanNumber()
	}

	switch ch {
	case eof:
		return TokenEOF, ""
	case '{':
		return TokenLCurlyBrace, string(ch)
	case '(':
		return TokenLParen, string(ch)
	case '[':
		return TokenLBracket, string(ch)
	case '}':
		return TokenRCurlyBrace, string(ch)
	case ')':
		return TokenRParen, string(ch)
	case ']':
		return TokenRBracket, string(ch)
	case ',':
		return TokenComma, string(ch)
	case '#':
		return s.scanComment()
	case '/':
		s.unread()
		return s.scanRegex()
	case '"':
		s.unread()
		return s.scanString()
	}

	return TokenIllegal, string(ch)
}
