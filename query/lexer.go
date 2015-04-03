package query

import (
	"unicode"

	"github.com/lestrrat/go-lex"
)

const (
	ItemMatchAnyElement lex.ItemType = lex.ItemDefaultMax + 1 + iota
	ItemMatchAnyElementShortHand
	ItemElementName
	ItemClassNamePrefixDot
	ItemClassName
	ItemIDPrefixPound
	ItemID
)

func makeLexer(q string) lex.Lexer {
	return lex.NewStringLexer(q, lexStart)
}

func lexStart(l lex.Lexer) lex.LexFn {
	l.AcceptRun(" \t")

	// must start with an element specification
	switch r := l.Peek(); {
	case r == lex.EOF:
		l.Emit(lex.ItemEOF)
		return nil
	case r == '#':
		return lexIDSelector
	case r == '.':
		l.Emit(ItemMatchAnyElementShortHand)
		return lexClassName
	case r == '*':
		l.Next()
		l.Emit(ItemMatchAnyElement)
		return lexElementSpecSuffix
	case r == '-' || unicode.IsDigit(r) || unicode.IsLetter(r):
		return lexElementSpec
	default:
		l.EmitErrorf("expected element specification")
		return nil
	}
}

func acceptID(l lex.Lexer) bool {
	if ! l.AcceptAny("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return false
	}

	for {
		if l.Peek() == '\\' {
			if l.Peek() != '\\' {
				return false
			}
			if !l.AcceptAny(":.[],") {
				return false
			}
		}
		if !l.AcceptRun("0123456789-_abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			break
		}
	}
	return true
}

func lexIDSelector(l lex.Lexer) lex.LexFn {
	if l.Peek() != '#' {
		l.EmitErrorf("expected id")
		return nil
	}
	l.Next()
	l.Emit(ItemIDPrefixPound)

	if !acceptID(l) {
		l.EmitErrorf("expected id")
		return nil
	}

	l.Emit(ItemID)

	return lexStart
}

func lexElementSpecSuffix(l lex.Lexer) lex.LexFn {
	if l.Peek() == '.' {
		return lexClassName
	}

	return lexStart
}

func lexElementSpec(l lex.Lexer) lex.LexFn {
	if !acceptIdent(l) {
		l.EmitErrorf("expected element name (ident)")
		return nil
	}

	l.Emit(ItemElementName)

	return lexElementSpecSuffix
}

func lexClassName(l lex.Lexer) lex.LexFn {
	if l.Next() != '.' {
		l.EmitErrorf("expected class name")
		return nil
	}

	l.Emit(ItemClassNamePrefixDot)

	if !acceptIdent(l) {
		l.EmitErrorf("expected class name (ident)")
		return nil
	}

	l.Emit(ItemClassName)

	return lexStart
}

func acceptIdentPrefix(l lex.Lexer) bool {
	if l.Peek() == '-' {
		l.Next()
	}
	return true
}

func acceptNonASCII(l lex.Lexer) bool {
	if l.Peek() != '\\' {
		return false
	}
	l.Next()

	first := l.Next()
	if first != '2' && first != '3' {
		for i := 0; i < 2; i++ {
			l.Backup()
		}
		return false
	}
	for i := 0; i < 2; i++ {
		if !unicode.IsDigit(l.Next()) {
			for j := 0; j < i+3; j++ {
				l.Backup()
			}
			return false
		}
	}
	return true
}

func acceptUnicode(l lex.Lexer) bool {
	if l.Peek() != '\\' {
		return false
	}
	l.Next()

	if !l.AcceptAny("0123456789abcdef") {
		l.Backup()
		return false
	}

	for i := 0; i < 5; i++ {
		if !l.AcceptAny("0123456789abcdef") {
			break
		}
	}

	return true
}

func acceptEscape(l lex.Lexer) bool {
	if acceptUnicode(l) {
		return true
	}

	// two backslashes
	for i := 0; i < 2; i++ {
		if l.Peek() != '\\' {
			for j := 0; j < i; j++ {
				l.Backup()
			}
			return false
		}
		l.Next()
	}

	switch l.Peek() {
	case '\r', '\n', '\f', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f':
		l.Next()
		return true
	default:
		l.Backup()
		l.Backup()
		return false
	}
}

func acceptIdent(l lex.Lexer) bool {
	if !acceptIdentPrefix(l) {
		return false
	}

	if !l.AcceptRun("_abcdefghijklmnopqrstuvwxyz") && !acceptNonASCII(l) && !acceptEscape(l) {
		return false
	}

	for {
		if !l.AcceptRun("-_abcdefghijklmnopqrstuvwxyz") &&
			!acceptNonASCII(l) &&
			!acceptEscape(l) {
			break
		}
	}

	return true
}
