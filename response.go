package mechanize

import (
	"bytes"
	"errors"
	"io/ioutil"
	"mime"
	"net/http"

	"golang.org/x/net/html"
)

type Response struct {
	*http.Response
	base       string
	forms      []*Form
	isHTML     bool
	mechanize  *Mechanize
	parsedHTML *html.Node
	rawBody    []byte
}

func NewResponse(m *Mechanize, res *http.Response) *Response {
	r := &Response{
		Response:  res,
		mechanize: m,
	}

	r.parseHeaders()
	r.parseHTML()
	return r
}

func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

func (r *Response) IsError() bool {
	return r.StatusCode >= 500 && r.StatusCode < 600
}

func (r *Response) Base() string {
	return r.base
}

func (r *Response) Forms() []*Form {
	return r.forms
}

func (r *Response) RawBody() []byte {
	return r.rawBody
}

func (r *Response) parseHeaders() {
	ct := r.Header.Get("Content-Type")
	if mt, _, err := mime.ParseMediaType(ct); err == nil {
		if mt == "text/html" || mt == "text/xhtml" {
			r.isHTML = true
		}
	}
}

func (r *Response) IsHTML() bool {
	return r.isHTML
}

func (r *Response) parseHTML() error {
	if r.Body == nil {
		return errors.New("field 'Body' is nil")
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	r.rawBody = body

	doc, err := html.Parse(bytes.NewReader(body))
	defer r.Body.Close()
	if err != nil {
		return err
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "form":
				r.forms = append(r.forms, NewForm(r.mechanize, n))
			case "base":
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						r.base = attr.Val
						break
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return nil
}