package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ketsugi/curlew/internal/config"
	"github.com/ketsugi/curlew/internal/hook"
	"github.com/ketsugi/curlew/internal/run"
	"github.com/spf13/cobra"
)

var version = "0.3.1"

var forceAnalyze bool

func main() {
	os.Exit(mainRun())
}

func mainRun() int {
	rootCmd := &cobra.Command{
		Use:           "curlew [url-or-file]",
		Short:         "Inspect before you execute",
		Long:          "curlew — inspect before you execute. A safe wrapper for curl|bash.",
		Args:          cobra.MaximumNArgs(1),
		RunE:          execute,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	rootCmd.Flags().BoolVar(&forceAnalyze, "force-analyze", false, "Run AI analysis even if prompt injection patterns are detected")
	rootCmd.Flags().String("hook", "", "Output shell hook code for eval (zsh or bash)")
	rootCmd.Flags().Bool("init-config", false, "Write a default config template and exit")

	rootCmd.AddCommand(listCmd())

	rootCmd.SetVersionTemplate("curlew {{.Version}}\n")
	rootCmd.Version = version

	if err := rootCmd.Execute(); err != nil {
		if errors.Is(err, run.ErrInterrupted) {
			return 130
		}
		fmt.Fprintf(os.Stderr, "\033[1;31merror:\033[0m %s\n", err)
		return 1
	}
	return 0
}

func execute(cmd *cobra.Command, args []string) error {
	hookShell, _ := cmd.Flags().GetString("hook")
	if hookShell != "" {
		return emitHook(hookShell)
	}

	initConfig, _ := cmd.Flags().GetBool("init-config")
	if initConfig {
		return writeConfigTemplate()
	}

	if len(args) == 0 {
		return cmd.Help()
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[1;33mwarning:\033[0m failed to load config: %s\n", err)
		cfg = config.Defaults()
	}

	return run.Execute(run.Options{
		Target:       args[0],
		ForceAnalyze: forceAnalyze,
		SkipTTY:      os.Getenv("CURLEW_SKIP_TTY_CHECK") == "1",
		Config:       cfg,
	})
}

func emitHook(shell string) error {
	switch shell {
	case "zsh":
		fmt.Print(hook.ZshHook())
	case "bash":
		fmt.Print(hook.BashHook())
	default:
		return fmt.Errorf("Unsupported shell: %s (supported: zsh, bash)", shell)
	}
	return nil
}

func writeConfigTemplate() error {
	path := config.FilePath()
	if path == "" {
		return fmt.Errorf("cannot determine config path (no home directory)")
	}

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists: %s", path)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(path, []byte(config.Template), 0o644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Wrote config template to: %s\n", path)
	return nil
}
