package engine

import (
	"encoding/json"
	"net/http"
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
