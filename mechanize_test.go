package mechanize

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func ExampleMechanize() {
}

type testServer struct {
	*httptest.Server
	ReqCh chan *http.Request
}

func newTestServer(h http.Handler) *testServer {
	s := &testServer{
		Server: nil,
		ReqCh:  make(chan *http.Request, 256),
	}
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.ReqCh <- r
		h.ServeHTTP(w, r)
	})

	s.Server = httptest.NewServer(wrapped)
	return s
}

func (ts *testServer) BaseURL() string {
	return ts.URL
}

func (ts *testServer) URLFor(path string, data url.Values) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u := ts.URL + path
	if qs := data.Encode(); qs != "" {
		u = u + "?" + qs
	}
	return u
}

const (
	page1Content = `
<html>
<head>
	<title>Page1</title>
</head>
<body>
	<form action="/form1" method="POST">
		<input type="text" name="username">
		<input type="password" name="password">
		<input type="submit" value="Login">
	</form>
</body>
</html>`
)

func startTestServer(t *testing.T) *testServer {
	return newTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Access detected to %s", r.URL.Path)
		switch r.URL.Path {
		case "/page1":
			io.WriteString(w, page1Content)
		case "/form1":
			r.ParseForm()
			io.WriteString(w, r.Form.Encode())
		case "/cookie/check":
			n := r.FormValue("name")
			if n == "" {
				n = "mechanize.cookie"
			}

			c, err := r.Cookie(n)
			if err != nil {
				http.Error(w, "No such cookie", 500)
				return
			}

			if v := r.FormValue("value"); v != "" {
				if c.Value != v {
					http.Error(
						w,
						fmt.Sprintf("value does not match. got '%s', expected '%s'", c.Value, v),
						500,
					)
					return
				}
			}

			w.Header().Set("Content-Type", "text/plain")
			io.WriteString(w, "All Good!")
		case "/cookie/gimme":
			n := r.FormValue("name")
			if n == "" {
				n = "mechanize.cookie"
			}
			p := r.FormValue("path")
			if p == "" {
				p = "/"
			}
			v := r.FormValue("value")

			c := &http.Cookie{
				Name:  n,
				Value: v,
				Path:  p,
			}

			http.SetCookie(w, c)
			w.Header().Set("Content-Type", "text/plain")
			io.WriteString(w, "Hello, World!")
		case "/redirect":
			u := r.FormValue("url")
			switch u {
			case "self":
				u = "/redirect?url=self"
			case "":
				http.Error(w, "Bad redirect url", 500)
				return
			}
			w.Header().Set("Location", u)
			w.WriteHeader(302)
		default:
			http.Error(w, "Not Found", 404)
		}
	}))
}

func TestMechanizeGet(t *testing.T) {
	ts0 := startTestServer(t)
	defer ts0.Close()

	m := NewMechanize()

	// First request with absolute URL, and second one with relative.
	// the second one should inherit the URL from the previous request
	for _, u := range []string{ts0.URLFor("/page1", nil), "/page1"} {
		if err := m.Get(u); err != nil {
			t.Errorf("Failed to fetch url /page1: %s", err)
			return
		}

		req := m.LastRequest()
		if req == nil {
			t.Errorf("LastRequest returned nil")
			return
		}

		if req.Header.Get("User-Agent") != m.Agent {
			t.Errorf("Expected User-Agent %s, got %s", m.Agent, req.Header.Get("User-Agent"))
			return
		}

		res := m.LastResponse()
		if res == nil {
			t.Errorf("LastResponse returned nil")
			return
		}
		if !res.IsSuccess() {
			t.Errorf("Previous request failed: %d", res.StatusCode)
			return
		}
	}
}

func TestMaxRedirects(t *testing.T) {
	ts0 := startTestServer(t)
	defer ts0.Close()

	m := NewMechanize()

	var err error
	err = m.Get(ts0.URLFor("/redirect", url.Values{"url": []string{"self"}}))
	if err == nil {
		t.Errorf("Redirect loop should have happened")
		return
	}

	if !strings.Contains(err.Error(), "stopped after 10 redirects") {
		t.Errorf("Expected redirect stop error message, got %s", err)
		return
	}

	m.SetMaxRedirects(0)
	err = m.Get(ts0.URLFor("/redirect", url.Values{"url": []string{"self"}}))
	if err == nil {
		t.Errorf("Redirect loop should have happened")
		return
	}

	if !strings.Contains(err.Error(), "redirects not allowed") {
		t.Errorf("Expected redirect stop error message, got %s", err)
		return
	}
}

func TestCookie(t *testing.T) {
	ts0 := startTestServer(t)
	defer ts0.Close()

	m := NewMechanize()

	v := url.Values{}
	v.Add("name", "mechanize.cookie")
	v.Add("value", "hello!")
	u := ts0.URLFor("/cookie/gimme", v)
	if err := m.Get(u); err != nil {
		t.Errorf("Failed to fetch %s: %s", u, err)
		return
	}

	if err := m.LastError(); err != nil {
		t.Errorf("Previous request failed: %s", err)
		return
	}

	u = ts0.URLFor("/cookie/check", v)
	if err := m.Get(u); err != nil {
		t.Errorf("Failed to fetch %s: %s", u, err)
		return
	}

	if err := m.LastError(); err != nil {
		t.Errorf("Previous request fialed: %s", err)
	}
}

func TestFormParse(t *testing.T) {
	ts0 := startTestServer(t)
	defer ts0.Close()

	m := NewMechanize()

	u := ts0.URLFor("/page1", nil)
	if err := m.Get(u); err != nil {
		t.Errorf("Failed to fetch %s: %s", u, err)
		return
	}

	forms := m.LastResponse().Forms()
	if len(forms) != 1 {
		t.Errorf("Expected 1 form, got %d", len(forms))
		return
	}

	f := forms[0]
	f.SetValue("username", "johndoe")
	f.SetValue("password", "passw0rd")

	fv, err := f.FormValues()
	if err != nil {
		t.Errorf("failed to encode form values: %s", err)
		return
	}

	if fv.Get("username") != "johndoe" {
		t.Errorf("expected username to be 'johndoe', got '%s'", fv.Get("username"))
		return
	}

	if fv.Get("password") != "passw0rd" {
		t.Errorf("expected password to be 'passw0rd', got '%s'", fv.Get("password"))
		return
	}

	f.Submit()
	if err := m.LastError(); err != nil {
		t.Errorf("Previous request fialed: %s", err)
		return
	}

	t.Logf("%s", m.LastResponse().Response.Request.URL.String())
	buf := m.LastResponse().RawBody()
	if string(buf) != fv.Encode() {
		t.Errorf("Got something else from server: %s", buf)
		return
	}
}
