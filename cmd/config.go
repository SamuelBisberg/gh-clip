package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/SamuelBisberg/gh-clip/pkg/config"
)

func newConfigCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "View or change gh-clip settings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigList()
		},
	}
	c.AddCommand(newConfigListCmd(), newConfigGetCmd(), newConfigSetCmd())
	return c
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show all current settings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigList()
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Print the value of a single setting",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigGet(args[0])
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Change a setting and save it",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSet(args[0], strings.Join(args[1:], " "))
		},
	}
}

func runConfigList() error {
	cfg, theme, err := loadTheme()
	if err != nil {
		return err
	}

	path, err := config.Path()
	if err == nil {
		theme.Infof("Config file: %s", path)
	}
	for _, key := range config.Keys() {
		value, _ := cfg.Get(key)
		if value == "" {
			value = theme.Faint.Render("(unset)")
		}
		fmt.Printf("  %s = %s\n", theme.Bold.Render(key), value)
	}
	return nil
}

// runConfigGet prints a single raw value with no styling, since it's meant
// to be consumed by scripts (e.g. `gh clip config get share.ttl`).
func runConfigGet(key string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	value, err := cfg.Get(key)
	if err != nil {
		return err
	}
	fmt.Println(value)
	return nil
}

func runConfigSet(key, value string) error {
	cfg, theme, err := loadTheme()
	if err != nil {
		return err
	}
	if err := cfg.Set(key, value); err != nil {
		return err
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	theme.Successf("Set %s = %s", key, value)
	return nil
}
