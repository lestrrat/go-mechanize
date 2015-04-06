package mechanize

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

const version = "0.0.1"

type historyEnt struct {
	request  *http.Request
	response *Response
}

type Mechanize struct {
	history     []*historyEnt
	Agent       string
	Client      *http.Client
	CookieJar   *cookiejar.Jar
	Headers     http.Header
	SendReferer bool
}

func New() *Mechanize {
	cjar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	m := &Mechanize{
		Agent:     "go-mechanize (Mechanize)/" + version,
		CookieJar: cjar,
		Client:    &http.Client{},
		Headers:   http.Header{},
	}

	m.Client.Jar = m.CookieJar
	m.SetMaxRedirects(10)
	return m
}

func (m *Mechanize) SetMaxRedirects(howmany int) {
	m.Client.CheckRedirect = FollowRedirectsCallback(howmany)
}

// FollowRedirectsCallback is a function to be passed to http.Client 's
// CheckRedirect callback, allowing all redirects, up to m.MaxRedirects
func FollowRedirectsCallback(howmany int) func(r *http.Request, via []*http.Request) error {
	return func(r *http.Request, via []*http.Request) error {
		if howmany == 0 {
			return errors.New("redirects not allowed")
		}

		if len(via) > howmany {
			return fmt.Errorf("stopped after %d redirects", howmany)
		}
		return nil
	}
}

func (m *Mechanize) ResolveURL(u *url.URL) *url.URL {
	res := m.LastResponse()
	if res == nil {
		return u
	}

	// <base> in the content takes precedence
	if res.IsHTML() {
		if res.base != "" {
			parsed, err := url.Parse(res.base)
			if err == nil {
				return parsed.ResolveReference(u)
			}
		}
	}

	// finally, use the URL from the latest request
	return res.Request.URL.ResolveReference(u)
}

func (m *Mechanize) BuildRequest(method, u string, body io.Reader) (*http.Request, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, m.ResolveURL(parsed).String(), body)
	if err != nil {
		return nil, err
	}

	referer := ""
	if m.SendReferer && len(m.history) > 0 {
		referer = m.history[0].response.Request.URL.String()
	}

	req.Header = m.Headers
	if agent := m.Agent; agent != "" {
		req.Header.Add("User-Agent", agent)
	}

	if referer != "" {
		req.Header.Add("Referer", referer)
	}

	return req, nil
}

func (m *Mechanize) Get(u string) error {
	req, err := m.BuildRequest("GET", u, nil)
	if err != nil {
		return err
	}

	return m.SendRequest(req)
}

func (m *Mechanize) Post(u string, bodyType string, body io.Reader) error {
	req, err := m.BuildRequest("POST", u, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", bodyType)

	return m.SendRequest(req)
}

func (m *Mechanize) PostForm(u string, data url.Values) error {
	return m.Post(u, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

func (m *Mechanize) Head(u string) error {
	req, err := m.BuildRequest("HEAD", u, nil)
	if err != nil {
		return err
	}

	return m.SendRequest(req)
}

func (m *Mechanize) SendRequest(req *http.Request) error {
	res, err := m.Client.Do(req)
	if err != nil {
		hdr := http.Header{}
		hdr.Set("Content-Type", "text/plain")
		hdr.Set("X-Mechanize-Failure", err.Error())

		res = &http.Response{
			Status:        "500 Internal Server",
			StatusCode:    500,
			Proto:         "HTTP/1.0",
			ProtoMajor:    1,
			ProtoMinor:    0,
			Header:        hdr,
			ContentLength: 0,
			Body:          ioutil.NopCloser(strings.NewReader("")),
			Request:       req,
		}
	}

	m.history = append(m.history, &historyEnt{
		request:  req,
		response: NewResponse(m, res),
	})

	return err
}

// LastRequest returns the most recent *http.Request. If there are no
// request/response recorded, then the this method returns nil
func (m *Mechanize) LastRequest() *http.Request {
	if len(m.history) <= 0 {
		return nil
	}

	return m.history[len(m.history)-1].request
}

// LastResponse returns the most recent *Response. If there are no
// request/response recorded, then the this method returns nil
func (m *Mechanize) LastResponse() *Response {
	if len(m.history) <= 0 {
		return nil
	}

	return m.history[len(m.history)-1].response
}

func (m *Mechanize) LastError() error {
	r := m.LastResponse()
	if r == nil {
		return errors.New("No response available")
	}

	if failReason := r.Header.Get("X-Mechanize-Failure"); failReason != "" {
		return errors.New(failReason)
	}

	if r.IsError() {
		return errors.New(r.Status)
	}

	return nil
}
