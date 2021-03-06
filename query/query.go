package query

import (
	"io"

	"golang.org/x/net/html"
)

type Selection struct {
	Nodes []*html.Node
}

type ContextNode interface {
	find([]matcher) *Selection
}

type Document struct {
	root *html.Node
}

func NewDocument(r io.Reader) (*Document, error) {
	root, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	return &Document{
		root: root,
	}, nil
}

func (d *Document) Find(query string) *Selection {
	ms := CompileQuery(query)

	return &Selection{Nodes: MatchNodes(d.root, ms)}
}

func MatchNodes(n *html.Node, ms []matcher) []*html.Node {
	if len(ms) == 0 {
		return nil
	}

	ret := []*html.Node{}
	if ms[0].Match(n) {
		ms = ms[1:]
		ret = append(ret, n)
	}

	if len(ms) > 0 {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			ret = append(ret, MatchNodes(c, ms)...)
		}
	}

	return ret
}

type matchTagName string

func (m matchTagName) match(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == string(m)
}