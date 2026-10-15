package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/SamuelBisberg/gh-clip/pkg/gh"
	"github.com/SamuelBisberg/gh-clip/pkg/scaffold"
)

func newSweepCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sweep",
		Short: "Run the cleanup workflow now to delete expired clips and Gists",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSweep()
		},
	}
}

func runSweep() error {
	cfg, theme, err := loadTheme()
	if err != nil {
		return err
	}
	sess, err := newSession()
	if err != nil {
		return err
	}

	repoPath := fmt.Sprintf("repos/%s/%s", sess.owner, cfg.Storage.Repo)
	var repo struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := sess.rest.Do(http.MethodGet, repoPath, nil, &repo); err != nil {
		return hintInit(fmt.Errorf("looking up %s: %w", cfg.Storage.Repo, err))
	}

	workflow := scaffold.WorkflowFile
	body := gh.Body(map[string]string{"ref": repo.DefaultBranch})
	err = theme.RunWithSpinner("Triggering cleanup workflow", func() error {
		return sess.rest.Do(http.MethodPost, fmt.Sprintf("%s/actions/workflows/%s/dispatches", repoPath, workflow), body, nil)
	})
	if err != nil {
		return fmt.Errorf("triggering %s: %w", workflow, err)
	}

	theme.Successf("Cleanup workflow started")
	theme.Infof("Follow it with: gh run watch --repo %s/%s", sess.owner, cfg.Storage.Repo)
	return nil
}
