package query

import (
	"os"
	"testing"
)

func newDoc(f string) *Document {
	fn, err := os.Open(f)
	if err != nil {
		panic(err)
	}
	defer fn.Close()

	doc, err := NewDocument(fn)
	if err != nil {
		panic(err)
	}
	return doc
}

func TestQuery(t *testing.T) {
	d := newDoc("testdata/peco.html")
	s := d.Find("p.clone-options")
	expected := 1
	if s == nil {
		t.Errorf("Expected %d nodes, got nil", expected)
		return
	}
	if len(s.Nodes) != expected {
		t.Errorf("Expected %d nodes, got %d", expected, len(s.Nodes))
		return
	}

	s = d.Find("li.header-nav-item")
	expected = 4
	if s == nil {
		t.Errorf("Expected %d nodes, got nil", expected)
		return
	}

	if len(s.Nodes) != expected {
		t.Errorf("Expected %d nodes, got %d", expected, len(s.Nodes))

		for _, n := range s.Nodes {
			t.Logf("Tag: %s", n.Data)
			for _, attr := range n.Attr {
				if attr.Key != "class" {
					t.Logf("    Class: %s", attr.Val)
				}
			}
		}

		return
	}
}

func TestQueryID(t *testing.T) {
	d := newDoc("testdata/peco.html")
	s := d.Find("#start-of-content")
	if s == nil {
		t.Errorf("Expected 1 nodes, got nil")
		return
	}

	t.Logf("%#v", s.Nodes)
}
