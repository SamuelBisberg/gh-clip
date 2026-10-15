// Package scaffold creates the GitHub-side infrastructure gh-clip relies on:
// the private storage repo (with its retention policy and cleanup workflow)
// and the public viewer repo served by GitHub Pages.
package scaffold

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/SamuelBisberg/gh-clip/pkg/gh"
)

// Embedded file contents pushed by `gh clip init`.
var (
	//go:embed files/retention.json
	RetentionJSON []byte
	//go:embed files/cleanup.yml
	CleanupWorkflow []byte
	//go:embed files/viewer.html
	ViewerHTML []byte
)

// Paths of the scaffolded files inside their repositories.
const (
	RetentionPath = "retention.json"
	WorkflowFile  = "cleanup.yml"
	WorkflowPath  = ".github/workflows/" + WorkflowFile
	ViewerPath    = "index.html"
)

// EnsureRepo creates owner/name if it doesn't exist yet and reports whether
// it had to. The repo is initialized with a README so it has a default
// branch the contents API can write to.
func EnsureRepo(rest gh.REST, owner, name string, private bool) (created bool, err error) {
	err = rest.Do(http.MethodGet, fmt.Sprintf("repos/%s/%s", owner, name), nil, nil)
	if err == nil {
		return false, nil
	}
	if !gh.IsStatus(err, http.StatusNotFound) {
		return false, fmt.Errorf("checking repo %s/%s: %w", owner, name, err)
	}

	body := gh.Body(map[string]any{
		"name":        name,
		"private":     private,
		"auto_init":   true,
		"description": "Managed by gh-clip",
	})
	if err := rest.Do(http.MethodPost, "user/repos", body, nil); err != nil {
		return false, fmt.Errorf("creating repo %s/%s: %w", owner, name, err)
	}
	return true, nil
}

// PutFile creates or overwrites path in owner/repo on its default branch.
func PutFile(rest gh.REST, owner, repo, path string, content []byte, message string) error {
	url := fmt.Sprintf("repos/%s/%s/contents/%s", owner, repo, path)

	var existing struct {
		SHA string `json:"sha"`
	}
	if err := rest.Do(http.MethodGet, url, nil, &existing); err != nil && !gh.IsStatus(err, http.StatusNotFound) {
		return fmt.Errorf("checking %s: %w", path, err)
	}

	payload := map[string]any{
		"message": message,
		"content": base64.StdEncoding.EncodeToString(content),
	}
	if existing.SHA != "" {
		payload["sha"] = existing.SHA
	}
	if err := rest.Do(http.MethodPut, url, gh.Body(payload), nil); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// EnablePages turns on GitHub Pages for owner/repo, serving the default
// branch root. An already-enabled site is not an error.
func EnablePages(rest gh.REST, owner, repo string) error {
	body := gh.Body(map[string]any{"source": map[string]string{"branch": "main", "path": "/"}})
	err := rest.Do(http.MethodPost, fmt.Sprintf("repos/%s/%s/pages", owner, repo), body, nil)
	if err != nil && !gh.IsStatus(err, http.StatusConflict) && !gh.IsStatus(err, http.StatusUnprocessableEntity) {
		return fmt.Errorf("enabling GitHub Pages: %w", err)
	}
	return nil
}
