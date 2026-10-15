package cmd

import (
	"bytes"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/SamuelBisberg/gh-clip/internal/ghtest"
	"github.com/SamuelBisberg/gh-clip/pkg/gh"
	"github.com/SamuelBisberg/gh-clip/pkg/tui"
)

// fakeEnv isolates config, swaps in a fake GitHub client, and restores every
// seam afterwards.
func fakeEnv(t *testing.T, handler func(method, path string, body []byte) (any, error)) *ghtest.Fake {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	f := &ghtest.Fake{}
	f.Handler = func(method, path string, body []byte) (any, error) {
		if method == http.MethodGet && path == "user" {
			return map[string]string{"login": "me"}, nil
		}
		return handler(method, path, body)
	}

	oldREST, oldPipe, oldTTY, oldClip, oldBrowser, oldSecret := newREST, stdinIsPipe, stdoutIsTTY, copyToClipboard, openBrowser, setSecret
	newREST = func() (gh.REST, error) { return f, nil }
	t.Cleanup(func() {
		newREST, stdinIsPipe, stdoutIsTTY, copyToClipboard, openBrowser, setSecret = oldREST, oldPipe, oldTTY, oldClip, oldBrowser, oldSecret
	})
	return f
}

func TestNewRootCmd(t *testing.T) {
	Version = "v1.2.3"
	t.Cleanup(func() { Version = "dev" })

	root := NewRootCmd()
	if root.Use != "clip" || root.Version != "v1.2.3" {
		t.Errorf("Use=%q Version=%q", root.Use, root.Version)
	}
	if root.Flags().Lookup("share") == nil {
		t.Error("missing --share flag")
	}
	for _, name := range []string{"init", "sweep", "config"} {
		found := false
		for _, c := range root.Commands() {
			found = found || c.Name() == name
		}
		if !found {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestPipedInputIsPushed(t *testing.T) {
	f := fakeEnv(t, func(method, path string, body []byte) (any, error) {
		return map[string]any{"content": map[string]any{"sha": "s"}}, nil
	})
	stdinIsPipe = func() bool { return true }

	root := NewRootCmd()
	root.SetIn(strings.NewReader("DB_PASS=123\n"))
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	put := f.Calls[len(f.Calls)-1]
	if put.Method != http.MethodPut || !strings.Contains(put.Path, "repos/me/gh-clip-storage/contents/clips/") {
		t.Errorf("last call = %s %s", put.Method, put.Path)
	}
}

func TestPushMissingRepoHintsInit(t *testing.T) {
	fakeEnv(t, func(method, path string, body []byte) (any, error) {
		return nil, &api.HTTPError{StatusCode: http.StatusNotFound}
	})
	stdinIsPipe = func() bool { return true }

	root := NewRootCmd()
	root.SetIn(strings.NewReader("x"))
	err := root.Execute()
	if !errors.Is(err, errNotInitialized) {
		t.Errorf("err = %v, want errNotInitialized", err)
	}
}

func TestEmptyInputRejected(t *testing.T) {
	fakeEnv(t, func(string, string, []byte) (any, error) { return nil, nil })
	stdinIsPipe = func() bool { return true }

	root := NewRootCmd()
	root.SetIn(strings.NewReader("\n\n"))
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("err = %v", err)
	}
}

func TestShareCreatesGist(t *testing.T) {
	f := fakeEnv(t, func(method, path string, body []byte) (any, error) {
		return map[string]string{"id": "abc123"}, nil
	})
	stdinIsPipe = func() bool { return true }

	root := NewRootCmd()
	root.SetArgs([]string{"--share"})
	root.SetIn(strings.NewReader("https://example.com\n"))
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	post := f.Calls[len(f.Calls)-1]
	if post.Method != http.MethodPost || post.Path != "gists" {
		t.Errorf("last call = %s %s", post.Method, post.Path)
	}
	if !bytes.Contains(post.Body, []byte(`"public":false`)) || !bytes.Contains(post.Body, []byte("gh-clip-share_")) {
		t.Errorf("body = %s", post.Body)
	}
}

func TestShareWithoutInput(t *testing.T) {
	fakeEnv(t, func(string, string, []byte) (any, error) { return nil, nil })
	stdinIsPipe = func() bool { return false }

	root := NewRootCmd()
	root.SetArgs([]string{"--share"})
	if err := root.Execute(); !errors.Is(err, errShareNeedsInput) {
		t.Errorf("err = %v", err)
	}
}

func TestPullRequiresTerminal(t *testing.T) {
	fakeEnv(t, func(string, string, []byte) (any, error) { return nil, nil })
	stdinIsPipe = func() bool { return false }
	stdoutIsTTY = func() bool { return false }

	if err := NewRootCmd().Execute(); err == nil || !strings.Contains(err.Error(), "terminal") {
		t.Errorf("err = %v", err)
	}
}

func TestSweepDispatchesWorkflow(t *testing.T) {
	f := fakeEnv(t, func(method, path string, body []byte) (any, error) {
		if method == http.MethodGet {
			return map[string]string{"default_branch": "main"}, nil
		}
		return nil, nil
	})

	root := NewRootCmd()
	root.SetArgs([]string{"sweep"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	last := f.Calls[len(f.Calls)-1]
	want := "repos/me/gh-clip-storage/actions/workflows/cleanup.yml/dispatches"
	if last.Method != http.MethodPost || last.Path != want || !bytes.Contains(last.Body, []byte(`"ref":"main"`)) {
		t.Errorf("last call = %s %s %s", last.Method, last.Path, last.Body)
	}
}

func TestInit(t *testing.T) {
	f := fakeEnv(t, func(method, path string, body []byte) (any, error) {
		if method == http.MethodGet {
			return nil, &api.HTTPError{StatusCode: http.StatusNotFound}
		}
		return nil, nil
	})
	var opened, gotRepo, gotName, gotValue string
	openBrowser = func(u string) error { opened = u; return nil }
	promptSecret = func(string) (string, error) { return " tok \n", nil }
	setSecret = func(repo, name, value string) error { gotRepo, gotName, gotValue = repo, name, value; return nil }
	t.Cleanup(func() { promptSecret = tui.PromptSecret })

	root := NewRootCmd()
	root.SetArgs([]string{"init"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(opened, "scopes=gist") {
		t.Errorf("opened %q", opened)
	}
	if gotRepo != "me/gh-clip-storage" || gotName != "GIST_TOKEN" || gotValue != "tok" {
		t.Errorf("secret = %s %s %q", gotRepo, gotName, gotValue)
	}

	var puts, pages []string
	for _, c := range f.Calls {
		if c.Method == http.MethodPut {
			puts = append(puts, c.Path)
		}
		if strings.HasSuffix(c.Path, "/pages") {
			pages = append(pages, c.Path)
		}
	}
	wantPuts := []string{
		"repos/me/gh-clip-storage/contents/retention.json",
		"repos/me/gh-clip-storage/contents/.github/workflows/cleanup.yml",
		"repos/me/gh-clip-storage/contents/README.md",
		"repos/me/gh-clip-viewer/contents/index.html",
		"repos/me/gh-clip-viewer/contents/README.md",
	}
	if !slices.Equal(puts, wantPuts) {
		t.Errorf("puts = %v, want %v", puts, wantPuts)
	}
	if len(pages) != 1 || pages[0] != "repos/me/gh-clip-viewer/pages" {
		t.Errorf("pages = %v", pages)
	}
}
