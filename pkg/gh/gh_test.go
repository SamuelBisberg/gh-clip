package gh

import (
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

func TestIsStatus(t *testing.T) {
	err := &api.HTTPError{StatusCode: http.StatusNotFound}
	if !IsStatus(err, 404) {
		t.Error("IsStatus(404 error, 404) = false")
	}
	if IsStatus(err, 500) {
		t.Error("IsStatus(404 error, 500) = true")
	}
	if IsStatus(errors.New("boom"), 404) {
		t.Error("IsStatus(plain error) = true")
	}
}

func TestBody(t *testing.T) {
	got, _ := io.ReadAll(Body(map[string]any{"a": 1}))
	if string(got) != `{"a":1}` {
		t.Errorf("Body = %s", got)
	}
}
