// Package gist creates the short-lived secret Gists used to hand a clip to
// a phone, and builds the viewer URL that opens them.
package gist

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/SamuelBisberg/gh-clip/pkg/config"
	"github.com/SamuelBisberg/gh-clip/pkg/gh"
)

// DescriptionPrefix marks Gists owned by gh-clip. The cleanup workflow only
// ever touches Gists whose description starts with it.
const DescriptionPrefix = "gh-clip-share_"

// FileName is the single file inside every shared Gist.
const FileName = "clip.txt"

// Description encodes the expiry (unix seconds) into the Gist description,
// which is what the cleanup workflow reads back.
func Description(expires time.Time) string {
	return DescriptionPrefix + strconv.FormatInt(expires.Unix(), 10)
}

// ParseExpiry is the inverse of Description.
func ParseExpiry(desc string) (time.Time, bool) {
	rest, ok := strings.CutPrefix(desc, DescriptionPrefix)
	if !ok {
		return time.Time{}, false
	}
	n, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(n, 0), true
}

// ViewerURL is the GitHub Pages URL that renders the "tap to copy" page for
// a Gist.
func ViewerURL(owner, id string) string {
	return fmt.Sprintf("https://%s.github.io/%s/?gist=%s", owner, config.ViewerRepo, id)
}

// Create makes a secret Gist holding text that expires at the given time,
// returning its ID.
func Create(rest gh.REST, text string, expires time.Time) (string, error) {
	body := gh.Body(map[string]any{
		"description": Description(expires),
		"public":      false,
		"files":       map[string]any{FileName: map[string]string{"content": text}},
	})
	var resp struct {
		ID string `json:"id"`
	}
	if err := rest.Do(http.MethodPost, "gists", body, &resp); err != nil {
		return "", fmt.Errorf("creating gist: %w", err)
	}
	return resp.ID, nil
}
