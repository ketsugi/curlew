package main

import (
	"fmt"
	"os"

	"github.com/ketsugi/curlew/internal/hook"
	"github.com/ketsugi/curlew/internal/run"
	"github.com/spf13/cobra"
)

var version = "0.2.2"

var forceAnalyze bool

func main() {
	rootCmd := &cobra.Command{
		Use:   "curlew [url-or-file]",
		Short: "Inspect before you execute",
		Long:  "curlew — inspect before you execute. A safe wrapper for curl|bash.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  execute,
		// Silence cobra's default error/usage printing so we control output.
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	rootCmd.Flags().BoolVar(&forceAnalyze, "force-analyze", false, "Run AI analysis even if prompt injection patterns are detected")
	rootCmd.Flags().String("hook", "", "Output shell hook code for eval (zsh or bash)")

	rootCmd.SetVersionTemplate("curlew {{.Version}}\n")
	rootCmd.Version = version

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "\033[1;31merror:\033[0m %s\n", err)
		os.Exit(1)
	}
}

func execute(cmd *cobra.Command, args []string) error {
	hookShell, _ := cmd.Flags().GetString("hook")
	if hookShell != "" {
		return emitHook(hookShell)
	}

	if len(args) == 0 {
		return cmd.Help()
	}

	return run.Execute(run.Options{
		Target:       args[0],
		ForceAnalyze: forceAnalyze,
		SkipTTY:      os.Getenv("CURLEW_SKIP_TTY_CHECK") == "1",
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
