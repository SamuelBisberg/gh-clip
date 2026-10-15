// Package gh is a thin layer over go-gh's REST client: a small interface so
// the rest of gh-clip can be tested against a fake, plus a few helpers for
// the GitHub API calls every command needs.
package gh

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/cli/go-gh/v2/pkg/api"
)

// REST is the subset of *api.RESTClient that gh-clip uses.
type REST interface {
	Do(method, path string, body io.Reader, response any) error
}

// NewREST returns a REST client authenticated as the user's `gh auth login`
// session.
func NewREST() (REST, error) {
	c, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("creating GitHub client (run `gh auth login`?): %w", err)
	}
	return c, nil
}

// Login returns the authenticated user's login.
func Login(c REST) (string, error) {
	var u struct {
		Login string `json:"login"`
	}
	if err := c.Do(http.MethodGet, "user", nil, &u); err != nil {
		return "", fmt.Errorf("fetching current user: %w", err)
	}
	return u.Login, nil
}

// Body JSON-encodes v as a request body.
func Body(v any) io.Reader {
	data, _ := json.Marshal(v) // only ever called with maps/structs of plain values
	return bytes.NewReader(data)
}

// IsStatus reports whether err is a GitHub API error with the given HTTP
// status code.
func IsStatus(err error, code int) bool {
	var he *api.HTTPError
	return errors.As(err, &he) && he.StatusCode == code
}
