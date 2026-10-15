package store

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/SamuelBisberg/gh-clip/internal/ghtest"
)

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

func TestNewNameRoundTrip(t *testing.T) {
	now := time.Unix(1700000000, 0)
	name := NewName(now)
	if !strings.HasPrefix(name, "1700000000-") || !strings.HasSuffix(name, ".txt") {
		t.Fatalf("NewName = %q", name)
	}
	got, ok := ParseCreated(name)
	if !ok || !got.Equal(now) {
		t.Errorf("ParseCreated(%q) = %v, %v", name, got, ok)
	}
	if _, ok := ParseCreated("garbage.txt"); ok {
		t.Error("ParseCreated accepted garbage")
	}
	if NewName(now) == name {
		t.Error("NewName is not unique")
	}
}

func TestPut(t *testing.T) {
	f := &ghtest.Fake{Handler: func(method, path string, body []byte) (any, error) {
		return map[string]any{"content": map[string]any{"sha": "abc"}}, nil
	}}
	c, err := New(f, "me", "store").Put("DB_PASS=123", time.Unix(1700000000, 0))
	if err != nil {
		t.Fatal(err)
	}
	call := f.Calls[0]
	if call.Method != http.MethodPut || !strings.HasPrefix(call.Path, "repos/me/store/contents/clips/1700000000-") {
		t.Errorf("unexpected call %s %s", call.Method, call.Path)
	}
	var sent map[string]string
	_ = json.Unmarshal(call.Body, &sent)
	if sent["content"] != b64("DB_PASS=123") {
		t.Errorf("content = %q", sent["content"])
	}
	if c.SHA != "abc" || c.Content != "DB_PASS=123" {
		t.Errorf("clip = %+v", c)
	}
}

func TestRecent(t *testing.T) {
	f := &ghtest.Fake{Handler: func(method, path string, body []byte) (any, error) {
		switch path {
		case "repos/me/store/contents/clips":
			return []map[string]any{
				{"name": "100-a.txt"}, {"name": "300-c.txt"}, {"name": "200-b.txt"}, {"name": ".gitkeep"},
			}, nil
		default:
			name := path[strings.LastIndex(path, "/")+1:]
			return map[string]any{"name": name, "sha": "sha-" + name, "encoding": "base64", "content": b64("body of " + name)}, nil
		}
	}}
	clips, err := New(f, "me", "store").Recent(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(clips) != 2 || clips[0].Name != "300-c.txt" || clips[1].Name != "200-b.txt" {
		t.Fatalf("clips = %+v", clips)
	}
	if clips[0].Content != "body of 300-c.txt" || !clips[0].Created.Equal(time.Unix(300, 0)) {
		t.Errorf("clip[0] = %+v", clips[0])
	}
}

func TestRecentMissingDir(t *testing.T) {
	f := &ghtest.Fake{Handler: func(method, path string, body []byte) (any, error) {
		return nil, &api.HTTPError{StatusCode: http.StatusNotFound}
	}}
	clips, err := New(f, "me", "store").Recent(10)
	if err != nil || len(clips) != 0 {
		t.Errorf("Recent = %v, %v; want empty, nil", clips, err)
	}
}

func TestDelete(t *testing.T) {
	f := &ghtest.Fake{Handler: func(method, path string, body []byte) (any, error) { return nil, nil }}
	if err := New(f, "me", "store").Delete(Clip{Name: "1-a.txt", SHA: "s"}); err != nil {
		t.Fatal(err)
	}
	call := f.Calls[0]
	if call.Method != http.MethodDelete || !strings.Contains(string(call.Body), `"sha":"s"`) {
		t.Errorf("call = %+v", call)
	}
}
