// Package main provides the CLI commands for ssgo.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	apicmd "github.com/ssgohq/ssgo/tool/internal/cmd/api"
	dbcmd "github.com/ssgohq/ssgo/tool/internal/cmd/db"
	rpccmd "github.com/ssgohq/ssgo/tool/internal/cmd/rpc"
	"github.com/ssgohq/ssgo/tool/internal/cmdctx"
	"github.com/ssgohq/ssgo/tool/internal/runner"
)

var (
	// verbose enables verbose output
	verbose bool

	// debug enables debug mode
	debug bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ss",
	Short: "ss - An all-in-one CLI for Go service development",
	Long: `ss (ssgo) is an all-in-one CLI tool for Go service development workflows.

It provides commands for:
  - api     Generate HTTP API services (Hertz-based)
  - rpc     Generate RPC services (Kitex-based)
  - db      Generate database models and repositories
  - run     Run development services

Configuration:
  - .ss.yaml    (project config)

For more information, visit: https://github.com/ssgohq/ssgo`,
	Version: Version,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Set debug mode environment variable
		if debug {
			os.Setenv("SS_DEBUG", "true")
		}
		if verbose {
			os.Setenv("SS_VERBOSE", "true")
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug mode")

	// Add built-in commands
	rootCmd.AddCommand(versionCmd)

	// Register subcommands
	rootCmd.AddCommand(apiCmd)
	rootCmd.AddCommand(rpcCmd)
	rootCmd.AddCommand(dbCmd)
	rootCmd.AddCommand(runCmd)
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  "Print detailed version information about ssgo CLI",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ssgo version %s\n", Version)
		fmt.Printf("  Git Commit: %s\n", GitCommit)
		fmt.Printf("  Build Date: %s\n", BuildDate)
		fmt.Printf("  Go Version: %s\n", GoVersion)
	},
}

// apiCmd wraps the api package commands
var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Generate HTTP API services (Hertz-based)",
	Long: `Generate Hertz HTTP servers from .api files.

Commands:
  new     Create a new .api file template
  gen     Generate Hertz code from .api file
  logic   Generate only logic files
  doc     Generate OpenAPI documentation

Examples:
  ss api new user
  ss api gen --api api/user.api -m github.com/org/user-api
  ss api logic --api api/user.api -m github.com/org/user-api
  ss api doc --api api/user.api --format yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := createContext(args)
		return apicmd.Execute(ctx)
	},
	DisableFlagParsing: true,
}

// rpcCmd wraps the rpc package commands
var rpcCmd = &cobra.Command{
	Use:   "rpc",
	Short: "Generate RPC services (Kitex-based)",
	Long: `Generate Kitex RPC server from .proto files.

Commands:
  new <name>   Create a new .proto file template
  gen          Generate Kitex code from .proto file
  model        Generate shared model (kitex_gen) only

Examples:
  ss rpc new user
  ss rpc gen --proto idl/user.proto --service UserService -m github.com/org/user-rpc
  ss rpc model --proto idl/user.proto -m github.com/org/common-pb -o common-pb`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := createContext(args)
		return rpccmd.Execute(ctx)
	},
	DisableFlagParsing: true,
}

// dbCmd wraps the db package commands
var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Generate database models and repositories",
	Long: `Generate database layer code for Go projects.

Commands:
  sqlc    Generate type-safe Go code from SQL queries using SQLC
  bun     Generate models and repositories using uptrace/bun ORM
  gorm    Generate models and repositories using GORM ORM
  parse   Parse database schema (for testing/debugging)

Examples:
  ss db sqlc init --dir ./my-rpc-service --migrations ../migrations
  ss db sqlc gen --dir ./my-rpc-service
  ss db bun gen --dsn 'postgres://user:pass@localhost:5432/mydb?sslmode=disable'
  ss db gorm gen --dsn 'postgres://user:pass@localhost:5432/mydb?sslmode=disable'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := createContext(args)
		return dbcmd.Execute(ctx)
	},
	DisableFlagParsing: true,
}

// Run command flags
var (
	runConfigFile string
	runNoWatch    bool
	runNoBuild    bool
	runNoTUI      bool
)

// runCmd wraps the run command
var runCmd = &cobra.Command{
	Use:   "run [services...]",
	Short: "Run development services with hot reload",
	Long: `Run and manage multiple micro-services with hot reload and TUI.

Commands:
  init    Scan project and generate run config in .ss.yaml

Examples:
  ss run                    # Run all services
  ss run api rpc            # Run specific services
  ss run --no-tui           # Run without TUI
  ss run init               # Generate run config`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle init subcommand
		if len(args) > 0 && args[0] == "init" {
			workDir, _ := os.Getwd()
			return runner.InitConfig(workDir)
		}

		// Load runner config from .ss.yaml
		workDir, _ := os.Getwd()
		runnerConfig, err := loadRunnerConfig(workDir)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Create runner options
		opts := runner.Options{
			Config:   runnerConfig,
			WorkDir:  workDir,
			Services: args,
			NoWatch:  runNoWatch,
			NoBuild:  runNoBuild,
			NoTUI:    runNoTUI,
			Verbose:  verbose || debug,
		}

		// Create and run runner
		r := runner.New(opts)
		return r.Run(context.Background())
	},
}

func init() {
	// Run command flags
	runCmd.Flags().StringVarP(&runConfigFile, "config", "c", "", "path to config file")
	runCmd.Flags().BoolVar(&runNoWatch, "no-watch", false, "disable file watching")
	runCmd.Flags().BoolVar(&runNoBuild, "no-build", false, "skip build step")
	runCmd.Flags().BoolVar(&runNoTUI, "no-tui", false, "disable TUI, use plain output")
}

// createContext creates a cmdctx.Context from command args
func createContext(args []string) *cmdctx.Context {
	ctx := cmdctx.New()
	ctx.Args = args
	ctx.WorkingDir, _ = os.Getwd()
	ctx.Debug = debug
	ctx.ParseArgs() // Parse flags from args
	return ctx
}

// loadRunnerConfig loads the runner configuration from .ss.yaml
func loadRunnerConfig(workDir string) (*runner.RunnerConfig, error) {
	configPath := filepath.Join(workDir, ".ss.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no .ss.yaml found, run 'ss run init' to generate one")
		}
		return nil, err
	}

	// Parse YAML
	var rawConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to parse .ss.yaml: %w", err)
	}

	// Extract run section
	runSection, ok := rawConfig["run"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no 'run' section found in .ss.yaml, run 'ss run init' to generate one")
	}

	// Parse runner config
	config := &runner.RunnerConfig{}

	// Parse build_delay
	if bd, ok := runSection["build_delay"].(string); ok {
		if d, err := time.ParseDuration(bd); err == nil {
			config.BuildDelay = d
		}
	}
	if config.BuildDelay == 0 {
		config.BuildDelay = 500 * time.Millisecond
	}

	// Parse kill_delay
	if kd, ok := runSection["kill_delay"].(string); ok {
		if d, err := time.ParseDuration(kd); err == nil {
			config.KillDelay = d
		}
	}
	if config.KillDelay == 0 {
		config.KillDelay = 5 * time.Second
	}

	// Parse services
	if services, ok := runSection["services"].([]interface{}); ok {
		for _, svc := range services {
			if svcMap, ok := svc.(map[string]interface{}); ok {
				svcConfig := runner.ServiceConfig{}

				if name, ok := svcMap["name"].(string); ok {
					svcConfig.Name = name
				}
				if dir, ok := svcMap["dir"].(string); ok {
					svcConfig.Dir = dir
				}
				if cmd, ok := svcMap["cmd"].(string); ok {
					svcConfig.Cmd = cmd
				}
				if run, ok := svcMap["run"].(string); ok {
					svcConfig.Run = run
				}
				if color, ok := svcMap["color"].(string); ok {
					svcConfig.Color = color
				}

				// Parse env
				if env, ok := svcMap["env"].([]interface{}); ok {
					for _, e := range env {
						if es, ok := e.(string); ok {
							svcConfig.Env = append(svcConfig.Env, es)
						}
					}
				}

				// Parse depends_on
				if deps, ok := svcMap["depends_on"].([]interface{}); ok {
					for _, d := range deps {
						if ds, ok := d.(string); ok {
							svcConfig.DependsOn = append(svcConfig.DependsOn, ds)
						}
					}
				}

				// Parse watch config
				if watch, ok := svcMap["watch"].(map[string]interface{}); ok {
					if include, ok := watch["include"].([]interface{}); ok {
						for _, i := range include {
							if is, ok := i.(string); ok {
								svcConfig.Watch.Include = append(svcConfig.Watch.Include, is)
							}
						}
					}
					if exclude, ok := watch["exclude"].([]interface{}); ok {
						for _, e := range exclude {
							if es, ok := e.(string); ok {
								svcConfig.Watch.Exclude = append(svcConfig.Watch.Exclude, es)
							}
						}
					}
				}

				config.Services = append(config.Services, svcConfig)
			}
		}
	}

	return config, nil
}
