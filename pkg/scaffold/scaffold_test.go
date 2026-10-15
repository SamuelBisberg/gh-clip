package scaffold

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/SamuelBisberg/gh-clip/internal/ghtest"
)

func notFound() error { return &api.HTTPError{StatusCode: http.StatusNotFound} }

func TestEmbeddedFiles(t *testing.T) {
	var r struct {
		TTL int `json:"clipTTLHours"`
	}
	if err := json.Unmarshal(RetentionJSON, &r); err != nil || r.TTL <= 0 {
		t.Errorf("retention.json invalid: %v (%+v)", err, r)
	}
	if !strings.Contains(string(CleanupWorkflow), "gh-clip-share_") {
		t.Error("cleanup.yml doesn't reference the gist description prefix")
	}
	if !strings.Contains(string(ViewerHTML), "navigator.clipboard.writeText") {
		t.Error("viewer.html missing clipboard write")
	}
}

func TestEnsureRepo(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		f := &ghtest.Fake{Handler: func(string, string, []byte) (any, error) { return nil, nil }}
		created, err := EnsureRepo(f, "me", "r", true)
		if err != nil || created || len(f.Calls) != 1 {
			t.Errorf("created=%v err=%v calls=%d", created, err, len(f.Calls))
		}
	})
	t.Run("missing", func(t *testing.T) {
		f := &ghtest.Fake{Handler: func(m, p string, b []byte) (any, error) {
			if m == http.MethodGet {
				return nil, notFound()
			}
			return nil, nil
		}}
		created, err := EnsureRepo(f, "me", "r", true)
		if err != nil || !created {
			t.Fatalf("created=%v err=%v", created, err)
		}
		if !strings.Contains(string(f.Calls[1].Body), `"private":true`) {
			t.Errorf("body = %s", f.Calls[1].Body)
		}
	})
}

func TestPutFile(t *testing.T) {
	t.Run("new file", func(t *testing.T) {
		f := &ghtest.Fake{Handler: func(m, p string, b []byte) (any, error) {
			if m == http.MethodGet {
				return nil, notFound()
			}
			return nil, nil
		}}
		if err := PutFile(f, "me", "r", "a.txt", []byte("hi"), "msg"); err != nil {
			t.Fatal(err)
		}
		var sent map[string]string
		_ = json.Unmarshal(f.Calls[1].Body, &sent)
		if sent["content"] != base64.StdEncoding.EncodeToString([]byte("hi")) || sent["sha"] != "" {
			t.Errorf("sent = %v", sent)
		}
	})
	t.Run("overwrite sends sha", func(t *testing.T) {
		f := &ghtest.Fake{Handler: func(m, p string, b []byte) (any, error) {
			if m == http.MethodGet {
				return map[string]string{"sha": "old"}, nil
			}
			return nil, nil
		}}
		if err := PutFile(f, "me", "r", "a.txt", []byte("hi"), "msg"); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(f.Calls[1].Body), `"sha":"old"`) {
			t.Errorf("body = %s", f.Calls[1].Body)
		}
	})
}

func TestEnablePagesIgnoresConflict(t *testing.T) {
	f := &ghtest.Fake{Handler: func(string, string, []byte) (any, error) {
		return nil, &api.HTTPError{StatusCode: http.StatusConflict}
	}}
	if err := EnablePages(f, "me", "r"); err != nil {
		t.Errorf("EnablePages = %v", err)
	}
}
