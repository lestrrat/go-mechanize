package query

import (
	"strings"

	"golang.org/x/net/html"
)

type matcher struct {
	id          string
	elementName string
	className   string
}

func CompileQuery(q string) []matcher {
	l := makeLexer(q)
	go l.Run()

	matchers := []matcher{}
	for item := range l.Items() {
		switch item.Type() {
		case ItemID:
			m := matcher{}
			m.id = item.Value()
			matchers = append(matchers, m)
		case ItemMatchAnyElement, ItemMatchAnyElementShortHand:
			m := matcher{}
			m.elementName = "*"
			matchers = append(matchers, m)
		case ItemElementName:
			m := matcher{}
			m.elementName = item.Value()
			matchers = append(matchers, m)
		case ItemClassName:
			matchers[len(matchers)-1].className = item.Value()
		}
	}

	return matchers
}

func (m matcher) Match(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}

	if id := m.id; id != "" {
		found := false
		for _, attr := range n.Attr {
			if attr.Key != "id" {
				continue
			}

			if attr.Val == id {
				found = true
				break
			}
		}
		return found
	}

	if name := m.elementName; name != "*" {
		if n.Data != name {
			return false
		}
	}

	if name := m.className; name != "" {
		found := false
		for _, attr := range n.Attr {
			if attr.Key != "class" {
				continue
			}

			if strings.Contains(attr.Val, name) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
