package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/SamuelBisberg/gh-clip/pkg/config"
	"github.com/SamuelBisberg/gh-clip/pkg/scaffold"
	"github.com/SamuelBisberg/gh-clip/pkg/tui"
)

// tokenURL opens GitHub's token form pre-filled with the one scope the
// cleanup workflow needs: deleting expired Gists.
const tokenURL = "https://github.com/settings/tokens/new?scopes=gist&description=gh-clip-gist-sweeper"

// promptSecret is a seam for the masked token prompt.
var promptSecret = tui.PromptSecret

// setSecret stores value as an Actions secret on owner/repo by piping it to
// `gh secret set`, so it never appears in the process list.
var setSecret = func(repo, name, value string) error {
	c := exec.Command("gh", "secret", "set", name, "--repo", repo)
	c.Stdin = strings.NewReader(value)
	if out, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("gh secret set: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// hasSecret reports whether the Actions secret name is already set on repo.
// Secret values can't be read back, so only existence is checked.
var hasSecret = func(repo, name string) (bool, error) {
	out, err := exec.Command("gh", "secret", "list", "--repo", repo, "--json", "name").Output()
	if err != nil {
		return false, fmt.Errorf("gh secret list: %w", err)
	}
	var secrets []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(out, &secrets); err != nil {
		return false, fmt.Errorf("parsing gh secret list: %w", err)
	}
	for _, s := range secrets {
		if s.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create the storage repo, viewer page, and cleanup workflow",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit()
		},
	}
}

func runInit() error {
	cfg, theme, err := loadTheme()
	if err != nil {
		return err
	}
	sess, err := newSession()
	if err != nil {
		return err
	}
	repo := cfg.Storage.Repo

	step := func(label string, fn func() error) error {
		if err := theme.RunWithSpinner(label, fn); err != nil {
			return fmt.Errorf("%s: %w", strings.ToLower(label[:1])+label[1:], err)
		}
		return nil
	}

	var created bool
	if err := step("Checking for storage repo "+repo, func() error {
		var err error
		created, err = scaffold.EnsureRepo(sess.rest, sess.owner, repo, true)
		return err
	}); err != nil {
		return err
	}
	if created {
		theme.Successf("Created private storage repo %s", repo)
	} else {
		theme.Infof("Storage repo %s already exists - keeping your existing files and adding any that are missing", repo)
	}

	// On an existing repo, never overwrite files: retention.json may have
	// been tuned by the user.
	put := scaffold.PutFile
	if !created {
		put = scaffold.PutFileIfMissing
	}
	if err := step("Pushing retention policy and cleanup workflow", func() error {
		if err := put(sess.rest, sess.owner, repo, scaffold.RetentionPath, scaffold.RetentionJSON, "clip: add retention policy"); err != nil {
			return err
		}
		if err := put(sess.rest, sess.owner, repo, scaffold.WorkflowPath, scaffold.CleanupWorkflow, "clip: add cleanup workflow"); err != nil {
			return err
		}
		return put(sess.rest, sess.owner, repo, scaffold.ReadmePath, scaffold.StorageReadme, "clip: update readme")
	}); err != nil {
		var hint string
		if strings.Contains(err.Error(), ".github/workflows") {
			hint = " (the token needs the `workflow` scope: gh auth refresh -s workflow)"
		}
		return fmt.Errorf("%w%s", err, hint)
	}

	if err := step("Publishing the mobile viewer page", func() error {
		if _, err := scaffold.EnsureRepo(sess.rest, sess.owner, config.ViewerRepo, false); err != nil {
			return err
		}
		if err := scaffold.PutFile(sess.rest, sess.owner, config.ViewerRepo, scaffold.ViewerPath, scaffold.ViewerHTML, "clip: update viewer"); err != nil {
			return err
		}
		if err := scaffold.PutFile(sess.rest, sess.owner, config.ViewerRepo, scaffold.ReadmePath, scaffold.ViewerReadme, "clip: update readme"); err != nil {
			return err
		}
		return scaffold.EnablePages(sess.rest, sess.owner, config.ViewerRepo)
	}); err != nil {
		return err
	}

	if err := setupGistToken(theme, sess.owner+"/"+repo); err != nil {
		return err
	}

	theme.Successf("gh-clip is ready")
	theme.Infof("Save a clip with: echo hello | gh clip")
	theme.Infof("Viewer page: https://%s.github.io/%s/ (Pages can take a minute to go live)", sess.owner, config.ViewerRepo)
	return nil
}

// setupGistToken walks the user through creating a gist-scoped token and
// stores it as the GIST_TOKEN secret the cleanup workflow reads. It does
// nothing if the secret already exists.
func setupGistToken(theme *tui.Theme, repo string) error {
	// If the lookup fails (e.g. no permission to list secrets), fall through
	// and ask for the token as before.
	if ok, err := hasSecret(repo, "GIST_TOKEN"); err == nil && ok {
		theme.Infof("GIST_TOKEN secret already set on %s - skipping token setup", repo)
		return nil
	}

	theme.Infof("The cleanup workflow needs a token with the `gist` scope to delete expired shares.")
	if err := openBrowser(tokenURL); err != nil {
		theme.Warningf("Couldn't open a browser. Create the token yourself at:\n  %s", tokenURL)
	}

	token, err := promptSecret("Paste the generated token")
	if err != nil {
		return err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("no token entered - re-run `gh clip init` to finish setup")
	}

	return theme.RunWithSpinner("Saving GIST_TOKEN secret", func() error {
		return setSecret(repo, "GIST_TOKEN", token)
	})
}
