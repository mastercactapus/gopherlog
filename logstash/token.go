package logstash

//go:generate stringer -type=Token token.go

type Token int

const (
	TokenIllegal Token = iota
	TokenEOF
	TokenUnexpectedEOF
	TokenWhitespace
	TokenIdentifier
	TokenAssignment
	TokenLCurlyBrace
	TokenLBracket
	TokenLParen
	TokenRParen
	TokenRCurlyBrace
	TokenRBracket
	TokenString
	TokenNumber
	TokenIf
	TokenElse
	TokenComment
	TokenComma
	TokenBool
	TokenEqRegex
	TokenNEqRegex
	TokenEq
	TokenNEq
	TokenLT
	TokenGT
	TokenLTE
	TokenGTE
	TokenAnd
	TokenOr
	TokenIn
	TokenNotIn
	TokenNand
	TokenXor
	TokenNot
	TokenNegate
	TokenRegex
)
