package gist

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/SamuelBisberg/gh-clip/internal/ghtest"
)

func TestDescriptionRoundTrip(t *testing.T) {
	exp := time.Unix(1700003600, 0)
	desc := Description(exp)
	if desc != "gh-clip-share_1700003600" {
		t.Errorf("Description = %q", desc)
	}
	got, ok := ParseExpiry(desc)
	if !ok || !got.Equal(exp) {
		t.Errorf("ParseExpiry = %v, %v", got, ok)
	}
	for _, bad := range []string{"", "other", DescriptionPrefix + "soon"} {
		if _, ok := ParseExpiry(bad); ok {
			t.Errorf("ParseExpiry(%q) ok", bad)
		}
	}
}

func TestViewerURL(t *testing.T) {
	want := "https://me.github.io/gh-clip-viewer/?gist=abc123"
	if got := ViewerURL("me", "abc123"); got != want {
		t.Errorf("ViewerURL = %q, want %q", got, want)
	}
}

func TestCreate(t *testing.T) {
	f := &ghtest.Fake{Handler: func(method, path string, body []byte) (any, error) {
		return map[string]string{"id": "g1"}, nil
	}}
	id, err := Create(f, "hello", time.Unix(5, 0))
	if err != nil || id != "g1" {
		t.Fatalf("Create = %q, %v", id, err)
	}
	var sent struct {
		Description string                       `json:"description"`
		Public      bool                         `json:"public"`
		Files       map[string]map[string]string `json:"files"`
	}
	_ = json.Unmarshal(f.Calls[0].Body, &sent)
	if sent.Public || sent.Description != "gh-clip-share_5" || sent.Files[FileName]["content"] != "hello" {
		t.Errorf("sent = %+v", sent)
	}
	if f.Calls[0].Path != "gists" {
		t.Errorf("path = %q", f.Calls[0].Path)
	}
}
