package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/shahadulhaider/restless/internal/engine"
	"github.com/shahadulhaider/restless/internal/importer"
	"github.com/shahadulhaider/restless/internal/parser"
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
		Short: "TUI HTTP client using .http files",
		Args:  cobra.MaximumNArgs(1),
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

	cmd := &cobra.Command{
		Use:   "run <file.http>",
		Short: "Execute all requests in a .http file headlessly",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			rootDir := filepath.Dir(filePath)

			reqs, err := parser.ParseFile(filePath)
			if err != nil {
				return fmt.Errorf("parsing %s: %w", filePath, err)
			}

			var envVars map[string]string
			if envName != "" {
				envFile, loadErr := parser.LoadEnvironments(rootDir)
				if loadErr == nil && envFile != nil {
					envVars, _ = parser.ResolveEnvironment(envFile, envName)
				}
			}
			if envVars == nil {
				envVars = make(map[string]string)
			}

			chainCtx := parser.NewChainContext()
			cookies := engine.NewCookieManager()

			for _, req := range reqs {
				resolved, _ := parser.ResolveRequest(&req, envVars, chainCtx)
				loaded, loadErr := parser.LoadFileBody(resolved, rootDir)
				if loadErr != nil {
					loaded = resolved
				}

				jar := cookies.JarForEnv(envName)
				resp, execErr := engine.ExecuteWithJar(loaded, jar)
				if execErr != nil {
					fmt.Fprintf(os.Stderr, "ERROR %s %s: %v\n", loaded.Method, loaded.URL, execErr)
					if failFast {
						return execErr
					}
					continue
				}

				if loaded.Name != "" {
					chainCtx.StoreResponse(loaded.Name, resp)
				}

				fmt.Fprintf(os.Stdout, "%s %s → %s (%dms)\n",
					loaded.Method, loaded.URL,
					resp.Status,
					resp.Timing.Total.Milliseconds())
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&envName, "env", "", "Environment name")
	cmd.Flags().BoolVar(&failFast, "fail-fast", false, "Stop on first error")
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
