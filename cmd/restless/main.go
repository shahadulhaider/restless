package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/shahadulhaider/restless/internal/importer"
	"github.com/shahadulhaider/restless/internal/runner"
	"github.com/shahadulhaider/restless/internal/tui"
)

var version = "dev"

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "restless [directory]",
		Short: "A terminal-native HTTP client with TUI — uses .http files",
		Long: `restless is a full-featured HTTP client that runs in your terminal.

It uses .http files — the same plain-text format supported by JetBrains IDEs
and VS Code REST Client. Browse requests in a TUI, send them, inspect responses,
and manage collections without leaving the terminal.

  restless .                    Launch TUI in current directory
  restless ./my-api             Launch TUI for a specific collection
  restless run api.http         Run requests headlessly (CI/CD)
  restless import openapi spec  Import from OpenAPI, Postman, Insomnia, Bruno

Press ? in the TUI for keyboard shortcuts, or F1 for context-sensitive help.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			abs, err := filepath.Abs(dir)
			if err != nil {
				return fmt.Errorf("resolving directory: %w", err)
			}
			return tui.RunApp(abs)
		},
	}

	root.AddCommand(importCmd(), runCmd(), versionCmd())
	return root
}

func importCmd() *cobra.Command {
	var envFile string
	var outputDir string

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import collections from other tools",
		Long: `Import API collections from Postman, Insomnia, Bruno, curl commands,
or OpenAPI/Swagger specs. Converts them to .http files.

Examples:
  restless import postman collection.json --output ./my-api
  restless import insomnia export.json --output ./api
  restless import bruno ./bruno-collection --output ./api
  restless import curl "curl -X POST https://api.example.com/users -H 'Content-Type: application/json'"
  restless import openapi swagger.yaml --output ./api`,
	}

	postman := &cobra.Command{
		Use:   "postman <collection.json>",
		Short: "Import a Postman collection v2.1",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := outputDir
			if out == "" {
				out = "."
			}
			abs, err := filepath.Abs(out)
			if err != nil {
				return fmt.Errorf("resolving output dir: %w", err)
			}
			if err := importer.ImportPostman(args[0], importer.ImportOptions{OutputDir: abs}); err != nil {
				return fmt.Errorf("importing collection: %w", err)
			}
			if envFile != "" {
				if err := importer.ImportPostmanEnv(envFile, abs); err != nil {
					return fmt.Errorf("importing environment: %w", err)
				}
			}
			fmt.Fprintf(os.Stdout, "Imported to %s\n", abs)
			return nil
		},
	}
	postman.Flags().StringVar(&envFile, "env", "", "Postman environment JSON file")
	postman.Flags().StringVar(&outputDir, "output", ".", "Output directory")

	insomnia := &cobra.Command{
		Use:   "insomnia <export.json>",
		Short: "Import an Insomnia v4 collection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := outputDir
			if out == "" {
				out = "."
			}
			abs, err := filepath.Abs(out)
			if err != nil {
				return fmt.Errorf("resolving output dir: %w", err)
			}
			if err := importer.ImportInsomnia(args[0], importer.ImportOptions{OutputDir: abs}); err != nil {
				return fmt.Errorf("importing collection: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Imported to %s\n", abs)
			return nil
		},
	}
	insomnia.Flags().StringVar(&outputDir, "output", ".", "Output directory")

	bruno := &cobra.Command{
		Use:   "bruno <collection-dir>",
		Short: "Import a Bruno collection directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := outputDir
			if out == "" {
				out = "."
			}
			abs, err := filepath.Abs(out)
			if err != nil {
				return fmt.Errorf("resolving output dir: %w", err)
			}
			if err := importer.ImportBruno(args[0], importer.ImportOptions{OutputDir: abs}); err != nil {
				return fmt.Errorf("importing collection: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Imported to %s\n", abs)
			return nil
		},
	}
	bruno.Flags().StringVar(&outputDir, "output", ".", "Output directory")

	curlCmd := &cobra.Command{
		Use:   "curl <command>",
		Short: "Import a curl command as a .http request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := outputDir
			if out == "" {
				out = "."
			}
			abs, err := filepath.Abs(out)
			if err != nil {
				return fmt.Errorf("resolving output dir: %w", err)
			}
			if err := importer.ImportCurl(args[0], importer.ImportOptions{OutputDir: abs}); err != nil {
				return fmt.Errorf("importing curl command: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Imported to %s\n", abs)
			return nil
		},
	}
	curlCmd.Flags().StringVar(&outputDir, "output", ".", "Output directory")

	openapi := &cobra.Command{
		Use:   "openapi <spec.json|spec.yaml>",
		Short: "Import an OpenAPI 3.x or Swagger 2.0 spec",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := outputDir
			if out == "" {
				out = "."
			}
			abs, err := filepath.Abs(out)
			if err != nil {
				return fmt.Errorf("resolving output dir: %w", err)
			}
			if err := importer.ImportOpenAPI(args[0], importer.ImportOptions{OutputDir: abs}); err != nil {
				return fmt.Errorf("importing spec: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Imported to %s\n", abs)
			return nil
		},
	}
	openapi.Flags().StringVar(&outputDir, "output", ".", "Output directory")

	cmd.AddCommand(postman, insomnia, bruno, curlCmd, openapi)
	return cmd
}

func runCmd() *cobra.Command {
	var envName string
	var failFast bool
	var insecure bool
	var proxyURL string
	var dataFile string
	var iterations int
	var delayMs int

	cmd := &cobra.Command{
		Use:   "run <file.http>",
		Short: "Execute all requests in a .http file headlessly",
		Long: `Execute all requests in a .http file without the TUI.

Useful for CI/CD pipelines and scripting. Supports environment variables,
request chaining, and response assertions (# @assert).

Use --data to run requests multiple times with different data from a CSV or JSON file.

Examples:
  restless run api.http
  restless run api.http --env production
  restless run api.http --env dev --fail-fast
  restless run api.http --data users.csv --env dev
  restless run api.http --data test-cases.json --iterations 5
  restless run api.http --insecure --proxy http://proxy:8080

Exit code is 1 if any request fails or any assertion fails.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := runner.RunConfig{
				FilePath:   args[0],
				EnvName:    envName,
				FailFast:   failFast,
				Insecure:   insecure,
				ProxyURL:   proxyURL,
				DataFile:   dataFile,
				Iterations: iterations,
				Delay:      time.Duration(delayMs) * time.Millisecond,
			}
			result, err := runner.Run(cfg)
			if err != nil {
				return err
			}
			if result.AnyFailed {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&envName, "env", "", "Environment name")
	cmd.Flags().BoolVar(&failFast, "fail-fast", false, "Stop on first error")
	cmd.Flags().BoolVar(&insecure, "insecure", false, "Skip TLS certificate verification")
	cmd.Flags().StringVar(&proxyURL, "proxy", "", "HTTP proxy URL (e.g. http://proxy:8080)")
	cmd.Flags().StringVar(&dataFile, "data", "", "CSV or JSON data file for parameterized runs")
	cmd.Flags().IntVar(&iterations, "iterations", 0, "Run only first N data rows (0 = all)")
	cmd.Flags().IntVar(&delayMs, "delay", 0, "Delay between iterations in milliseconds")
	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("restless", version)
		},
	}
}
