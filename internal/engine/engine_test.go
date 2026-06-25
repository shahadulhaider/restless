package engine

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shahadulhaider/restless/internal/model"
)

func TestEngineGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	req := &model.Request{Method: "GET", URL: srv.URL}
	resp, err := Execute(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, string(resp.Body), "ok")
	assert.Greater(t, resp.Timing.Total, time.Duration(0))
}

func TestEnginePOSTWithBody(t *testing.T) {
	var received []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		received = buf[:n]
		w.WriteHeader(201)
	}))
	defer srv.Close()

	req := &model.Request{
		Method: "POST",
		URL:    srv.URL,
		Headers: []model.Header{
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: `{"name":"test"}`,
	}
	resp, err := Execute(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)
	assert.Contains(t, string(received), "test")
}

func TestEngineDuplicateHeaders(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Values("X-Dup")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	req := &model.Request{
		Method: "GET",
		URL:    srv.URL,
		Headers: []model.Header{
			{Key: "X-Dup", Value: "one"},
			{Key: "X-Dup", Value: "two"},
		},
	}
	_, err := Execute(req)
	require.NoError(t, err)
	assert.Equal(t, []string{"one", "two"}, got, "duplicate request headers must both be sent")
}

func TestEngineNoRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	req := &model.Request{
		Method:   "GET",
		URL:      srv.URL + "/",
		Metadata: model.RequestMetadata{NoRedirect: true},
	}
	resp, err := Execute(req)
	require.NoError(t, err)
	assert.Equal(t, 302, resp.StatusCode)
}

func TestEngineInvalidProxyErrors(t *testing.T) {
	req := &model.Request{
		Method:   "GET",
		URL:      "http://127.0.0.1:9/get",
		Metadata: model.RequestMetadata{Proxy: "::bad::"},
	}
	_, err := Execute(req)
	require.Error(t, err, "an invalid @proxy URL must error instead of going direct")
	assert.Contains(t, err.Error(), "invalid proxy URL")
}

func TestEngineTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	req := &model.Request{
		Method:   "GET",
		URL:      srv.URL,
		Metadata: model.RequestMetadata{Timeout: 50 * time.Millisecond},
	}
	_, err := Execute(req)
	assert.Error(t, err)
}

func TestEngineNoCookieJarBypassesJar(t *testing.T) {
	var received []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("sess"); err == nil {
			received = append(received, c.Value)
		} else {
			received = append(received, "")
		}
		http.SetCookie(w, &http.Cookie{Name: "sess", Value: "v1", Path: "/"})
		w.WriteHeader(200)
	}))
	defer srv.Close()

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	_, err = ExecuteWithJar(&model.Request{Method: "GET", URL: srv.URL}, jar)
	require.NoError(t, err)

	_, err = ExecuteWithJar(&model.Request{
		Method:   "GET",
		URL:      srv.URL,
		Metadata: model.RequestMetadata{NoCookieJar: true},
	}, jar)
	require.NoError(t, err)

	_, err = ExecuteWithJar(&model.Request{Method: "GET", URL: srv.URL}, jar)
	require.NoError(t, err)

	require.Len(t, received, 3)
	assert.Equal(t, "", received[0], "first request has no cookie yet")
	assert.Equal(t, "", received[1], "@no-cookie-jar request must not send the stored cookie")
	assert.Equal(t, "v1", received[2], "a normal request still sends the stored cookie")
}

func TestEngineTimingPopulated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	req := &model.Request{Method: "GET", URL: srv.URL}
	resp, err := Execute(req)
	require.NoError(t, err)
	assert.Greater(t, resp.Timing.Total, time.Duration(0))
}
