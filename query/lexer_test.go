package query

import (
	"testing"

	"github.com/lestrrat/go-lex"
)

func TestLexer(t *testing.T) {
	tests := map[string][]lex.ItemType{
		"hello":                    {ItemElementName},
		"-hello":                   {ItemElementName},
		"-_hello_world":            {ItemElementName},
		"hello world":              {ItemElementName, ItemElementName},
		"hello.world":              {ItemElementName, ItemClassNamePrefixDot, ItemClassName},
		"hello.world bomdia.mundo": {ItemElementName, ItemClassNamePrefixDot, ItemClassName, ItemElementName, ItemClassNamePrefixDot, ItemClassName},
	}

	for input, expected := range tests {
		t.Logf("testing '%s'", input)
		l := makeLexer(input)
		go l.Run()

		for i := 0; i < len(expected); i++ {
			v := <-l.Items()
			if v == nil {
				t.Errorf("expected more tokens, but found end")
				return
			}
			if expected[i] != v.Type() {
				t.Errorf("expected %d, got %d", expected[i], v)
			}
		}
	}
}