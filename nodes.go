package mechanize

import (
	"errors"
	"net/url"

	"golang.org/x/net/html"
)

const (
	contentTypeFormUrlEncoded    = "application/x-www-form-urlencoded"
	contentTypeMultipartFormData = "multipart/form-data"
)

type FormField interface {
	RawNode() *html.Node
	Name() string
	Value() string
	SetValue(string)
}

type Input struct {
	*html.Node
}

func NewInput(n *html.Node) *Input {
	return &Input{
		Node: n,
	}
}

func (i Input) RawNode() *html.Node {
	return i.Node
}

func (i Input) Name() string {
	for _, attr := range i.Attr {
		if attr.Key == "name" {
			return attr.Val
		}
	}
	return ""
}

func (i Input) Value() string {
	for _, attr := range i.Attr {
		if attr.Key == "value" {
			return attr.Val
		}
	}
	return ""
}

func (i *Input) SetValue(v string) {
	for _, attr := range i.Attr {
		if attr.Key == "value" {
			attr.Val = v
			return
		}
	}
	// It's possible that we have no such attribute.
	// just create one in that case
	i.Attr = append(i.Attr, html.Attribute{Key: "value", Val: v})
}

type Form struct {
	*html.Node
	mechanize *Mechanize
	action    string
	enctype   string
	method    string
	fields    []FormField
}

func NewForm(m *Mechanize, n *html.Node) *Form {
	f := &Form{
		Node:      n,
		enctype:   contentTypeFormUrlEncoded,
		mechanize: m,
	}
	f.parse()
	return f
}

func (f *Form) parse() {
	for _, attr := range f.Attr {
		switch attr.Key {
		case "action":
			f.action = attr.Val
		case "method":
			f.method = attr.Val
		case "enctype":
			f.enctype = attr.Val
		}
	}

	var fn func(*html.Node)
	fn = func(n *html.Node) {
		// descend into children, and find out input elements
		// TODO: handle buttons and such also...
		if n.Type == html.ElementNode {
			if n.Data == "input" {
				f.fields = append(f.fields, NewInput(n))
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			fn(c)
		}
	}
	fn(f.Node)
}

func (f *Form) FindField(name string) (FormField, error) {
	for _, field := range f.fields {
		if field.Name() == name {
			return field, nil
		}
	}
	return nil, errors.New("field not found")
}

func (f *Form) SetValue(name, value string) error {
	field, err := f.FindField(name)
	if err != nil {
		return err
	}

	field.SetValue(value)
	return nil
}

func (f *Form) FormValues() (url.Values, error) {
	if f.enctype != contentTypeFormUrlEncoded {
		return nil, errors.New("form is not an 'application/x-www-form-urlencoded' enctype")
	}

	values := url.Values{}
	for _, n := range f.fields {
		if n.Name() == "" {
			continue
		}
		values.Set(n.Name(), n.Value())
	}
	return values, nil
}

func (f *Form) Submit() error {
	switch f.enctype {
	case contentTypeFormUrlEncoded:
		v, _ := f.FormValues()
		return f.mechanize.PostForm(f.action, v)
	case contentTypeMultipartFormData:
		panic("unimplemented")
	default:
		panic("unimplemented")
	}
}