package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/SamuelBisberg/gh-clip/pkg/gist"
	"github.com/SamuelBisberg/gh-clip/pkg/store"
	"github.com/SamuelBisberg/gh-clip/pkg/tui"
)

// recentLimit caps how many clips the picker loads.
const recentLimit = 50

// runPull opens the interactive picker over recent clips and copies the
// chosen one to the system clipboard.
func runPull() error {
	cfg, theme, err := loadTheme()
	if err != nil {
		return err
	}
	if !stdoutIsTTY() {
		return errors.New("nothing piped in and not a terminal - pipe text in to save a clip")
	}
	sess, err := newSession()
	if err != nil {
		return err
	}
	st := sess.store(cfg)

	var clips []store.Clip
	err = theme.RunWithSpinner("Loading clips", func() error {
		var err error
		clips, err = st.Recent(recentLimit)
		return err
	})
	if err != nil {
		return hintInit(err)
	}

	res, err := tui.RunList(theme, clips, tui.ListActions{
		Delete: st.Delete,
		Share: func(c store.Clip) (string, error) {
			id, err := gist.Create(sess.rest, c.Content, time.Now().Add(cfg.ShareTTL()))
			if err != nil {
				return "", err
			}
			return gist.ViewerURL(sess.owner, id), nil
		},
	})
	if err != nil {
		return fmt.Errorf("running picker: %w", err)
	}
	if res.Selected == nil {
		return nil
	}

	if err := copyToClipboard(res.Selected.Content); err != nil {
		theme.Warningf("Couldn't reach the system clipboard (%v) - printing the clip instead:", err)
		fmt.Println(res.Selected.Content)
		return nil
	}
	theme.Successf("Copied to clipboard")
	return nil
}
