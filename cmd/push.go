package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/SamuelBisberg/gh-clip/pkg/gist"
	"github.com/SamuelBisberg/gh-clip/pkg/tui"
)

var errShareNeedsInput = errors.New("--share needs piped input, e.g. `cat url.txt | gh clip --share`")

// readInput slurps piped text, dropping trailing newlines so that
// `echo x | gh clip` stores "x" rather than "x\n".
func readInput(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}
	text := strings.TrimRight(string(data), "\r\n")
	if strings.TrimSpace(text) == "" {
		return "", errors.New("nothing to save - input is empty")
	}
	return text, nil
}

// runPush stores piped text in the private storage repo.
func runPush(in io.Reader) error {
	cfg, theme, err := loadTheme()
	if err != nil {
		return err
	}
	text, err := readInput(in)
	if err != nil {
		return err
	}
	sess, err := newSession()
	if err != nil {
		return err
	}

	err = theme.RunWithSpinner("Saving clip", func() error {
		_, err := sess.store(cfg).Put(text, time.Now())
		return err
	})
	if err != nil {
		return hintInit(err)
	}

	theme.Successf("Saved clip to %s/%s", sess.owner, cfg.Storage.Repo)
	theme.Infof("Copy it on another device with: gh clip")
	return nil
}

// runShare publishes piped text as an expiring secret Gist and prints a QR
// code pointing at the mobile viewer.
func runShare(in io.Reader) error {
	cfg, theme, err := loadTheme()
	if err != nil {
		return err
	}
	text, err := readInput(in)
	if err != nil {
		return err
	}
	sess, err := newSession()
	if err != nil {
		return err
	}

	expires := time.Now().Add(cfg.ShareTTL())
	var id string
	err = theme.RunWithSpinner("Creating secret gist", func() error {
		var err error
		id, err = gist.Create(sess.rest, text, expires)
		return err
	})
	if err != nil {
		return err
	}

	url := gist.ViewerURL(sess.owner, id)
	theme.Successf("Shared! Scan to open on your phone:")
	fmt.Println()
	if err := tui.PrintQR(os.Stdout, url); err != nil {
		return fmt.Errorf("rendering QR code: %w", err)
	}
	fmt.Println()
	theme.Infof("%s", url)
	theme.Infof("Expires in %s", cfg.ShareTTL().Round(time.Minute))
	return nil
}
