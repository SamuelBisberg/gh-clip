// Package store reads and writes clips as plaintext files in the user's
// private GitHub storage repository, via the REST contents API.
package store

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SamuelBisberg/gh-clip/pkg/gh"
)

// Dir is the directory inside the storage repo that holds clips.
const Dir = "clips"

// Clip is one stored snippet. Names look like "<unix-ts>-<id>.txt", so
// they sort chronologically and carry their own creation time.
type Clip struct {
	Name    string
	SHA     string
	Created time.Time
	Content string
}

// Store talks to one owner/repo pair.
type Store struct {
	rest  gh.REST
	owner string
	repo  string
}

// New returns a Store for owner/repo.
func New(rest gh.REST, owner, repo string) *Store {
	return &Store{rest: rest, owner: owner, repo: repo}
}

func (s *Store) path(name string) string {
	return fmt.Sprintf("repos/%s/%s/contents/%s/%s", s.owner, s.repo, Dir, name)
}

// NewName builds a clip file name for the given time.
func NewName(now time.Time) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%d-%s.txt", now.Unix(), hex.EncodeToString(b))
}

// ParseCreated extracts the creation time from a clip file name.
func ParseCreated(name string) (time.Time, bool) {
	ts, _, ok := strings.Cut(name, "-")
	if !ok {
		return time.Time{}, false
	}
	n, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(n, 0), true
}

// Put stores text as a new clip.
func (s *Store) Put(text string, now time.Time) (Clip, error) {
	name := NewName(now)
	body := gh.Body(map[string]any{
		"message": "clip: add " + name,
		"content": base64.StdEncoding.EncodeToString([]byte(text)),
	})
	var resp struct {
		Content struct {
			SHA string `json:"sha"`
		} `json:"content"`
	}
	if err := s.rest.Do(http.MethodPut, s.path(name), body, &resp); err != nil {
		return Clip{}, fmt.Errorf("storing clip: %w", err)
	}
	return Clip{Name: name, SHA: resp.Content.SHA, Created: now, Content: text}, nil
}

type contentEntry struct {
	Name     string `json:"name"`
	SHA      string `json:"sha"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

func (e contentEntry) decode() (string, error) {
	if e.Encoding != "base64" {
		return e.Content, nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(e.Content, "\n", ""))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// Get fetches one clip by file name, including its content.
func (s *Store) Get(name string) (Clip, error) {
	var e contentEntry
	if err := s.rest.Do(http.MethodGet, s.path(name), nil, &e); err != nil {
		return Clip{}, fmt.Errorf("fetching %s: %w", name, err)
	}
	text, err := e.decode()
	if err != nil {
		return Clip{}, fmt.Errorf("decoding %s: %w", name, err)
	}
	created, _ := ParseCreated(e.Name)
	return Clip{Name: e.Name, SHA: e.SHA, Created: created, Content: text}, nil
}

// maxFetchers bounds how many clip bodies are downloaded concurrently.
const maxFetchers = 8

// Recent returns up to limit of the newest clips, newest first, with
// content loaded. A missing clips/ directory is an empty list, not an error.
func (s *Store) Recent(limit int) ([]Clip, error) {
	var entries []contentEntry
	err := s.rest.Do(http.MethodGet, fmt.Sprintf("repos/%s/%s/contents/%s", s.owner, s.repo, Dir), nil, &entries)
	if gh.IsStatus(err, http.StatusNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("listing clips: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name, ".txt") {
			names = append(names, e.Name)
		}
	}
	// Names start with a fixed-width-for-centuries unix timestamp, so a
	// reverse lexical sort is newest-first.
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	if limit > 0 && len(names) > limit {
		names = names[:limit]
	}

	clips := make([]Clip, len(names))
	errs := make([]error, len(names))
	sem := make(chan struct{}, maxFetchers)
	var wg sync.WaitGroup
	for i, name := range names {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			clips[i], errs[i] = s.Get(name)
		}()
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return clips, nil
}

// Delete removes a clip from the repository.
func (s *Store) Delete(c Clip) error {
	body := gh.Body(map[string]any{
		"message": "clip: delete " + c.Name,
		"sha":     c.SHA,
	})
	if err := s.rest.Do(http.MethodDelete, s.path(c.Name), body, nil); err != nil {
		return fmt.Errorf("deleting %s: %w", c.Name, err)
	}
	return nil
}
