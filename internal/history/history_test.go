package history

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shahadulhaider/restless/internal/model"
)

func makeReq(method, url string) *model.Request {
	return &model.Request{Method: method, URL: url}
}

func makeResp(code int, body string) *model.Response {
	return &model.Response{
		StatusCode: code,
		Status:     "OK",
		Body:       []byte(body),
		Timestamp:  time.Now(),
	}
}

func TestSaveAndList(t *testing.T) {
	dir := t.TempDir()
	req := makeReq("GET", "https://example.com/api")
	resp1 := makeResp(200, `{"v":1}`)
	resp2 := makeResp(200, `{"v":2}`)

	require.NoError(t, Save(dir, req, resp1, "dev"))
	require.NoError(t, Save(dir, req, resp2, "dev"))

	entries, err := List(dir, req)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.True(t, entries[0].Timestamp.After(entries[1].Timestamp), "newest first")
}

func TestHistoryDirAutoCreated(t *testing.T) {
	dir := t.TempDir()
	histDir := historyDir(dir)
	os.RemoveAll(histDir)

	req := makeReq("POST", "https://example.com/submit")
	resp := makeResp(201, `{"id":42}`)
	require.NoError(t, Save(dir, req, resp, ""))

	_, err := os.Stat(histDir)
	assert.NoError(t, err, "history dir should be created")
}

func TestListEmptyWhenNone(t *testing.T) {
	dir := t.TempDir()
	req := makeReq("GET", "https://example.com/none")
	entries, err := List(dir, req)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestDiff(t *testing.T) {
	req := makeReq("GET", "https://example.com/api")
	a := &HistoryEntry{
		Request:  req,
		Response: &model.Response{StatusCode: 200, Status: "OK", Body: []byte("line1\nline2\n")},
	}
	b := &HistoryEntry{
		Request:  req,
		Response: &model.Response{StatusCode: 404, Status: "Not Found", Body: []byte("line1\nline3\n")},
	}
	diff := Diff(a, b)
	assert.Contains(t, diff, "- Status: 200")
	assert.Contains(t, diff, "+ Status: 404")
	assert.Contains(t, diff, "- line2")
	assert.Contains(t, diff, "+ line3")
}
