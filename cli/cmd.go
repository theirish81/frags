package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"cloud.google.com/go/auth/credentials"
	"github.com/spf13/cobra"
	"github.com/theirish81/frags"
	"github.com/theirish81/frags/gemini"
	"github.com/theirish81/frags/ollama"
	"google.golang.org/genai"
	"gopkg.in/yaml.v3"
)

// supported output formats
const (
	formatTemplate = "template"
	formatYAML     = "yaml"
	formatJSON     = "json"
)

var (
	format       string
	output       string
	templatePath string
	params       []string
)

var rootCmd = cobra.Command{
	Use:   filepath.Base(os.Args[0]),
	Short: "CLI for Frags. Run a Frags session from a YAML file.",
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a session",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// validate flags and input
		if err := validateRunArgs(args); err != nil {
			cmd.PrintErrln(err)
			return
		}

		mcpConfig := frags.McpConfig{}
		if data, err := os.ReadFile("mcp.json"); err == nil {
			if err := json.Unmarshal(data, &mcpConfig); err != nil {
				cmd.PrintErrln(err)
				return
			}
		}

		// read session YAML
		data, err := os.ReadFile(args[0])
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		// build session manager from YAML
		sm := frags.NewSessionManager()
		if err := sm.FromYAML(data); err != nil {
			cmd.PrintErrln(err)
			return
		}

		dir := filepath.Dir(args[0])
		workers := cfg.ParallelWorkers
		if workers <= 0 {
			workers = 1
		}
		var ai frags.Ai
		switch cfg.guessAi() {
		case engineGemini:
			client, err := newGeminiClient()
			if err != nil {
				cmd.PrintErrln(err)
				return
			}
			ai = gemini.NewAI(client)
		case engineOllama:
			ai = ollama.NewAI(cfg.OllamaBaseURL, cfg.OllamaModel)
		default:
			cmd.PrintErrln("No AI is fully configured. Check your .env file")
			return
		}
		for name, mcpServer := range mcpConfig.McpServers {
			tool := frags.NewMcpTool(name)
			if err := tool.Connect(context.Background(), mcpServer); err != nil {
				cmd.PrintErrln(err)
				return
			}
			functions, err := tool.AsFunctions(context.Background())
			if err != nil {
				cmd.PrintErrln(err)
				return
			}
			ai.SetFunctions(functions)
		}
		ch := make(chan frags.ProgressMessage)
		go func() {
			for msg := range ch {
				fmt.Print(msg.Action, ":\t", msg.Session, "/", msg.Phase)
				if msg.Error != nil {
					fmt.Print("\tERROR: ", msg.Error.Error())
				}
				fmt.Println()
			}
		}()
		runner := frags.NewRunner[frags.ProgMap](
			sm,
			frags.NewFileResourceLoader(dir),
			ai,
			frags.WithSessionWorkers(workers),
			frags.WithLogger(slog.Default()),
			frags.WithProgressChannel(ch),
		)

		paramsMap, err := sliceToMap(params)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		// execute
		result, err := runner.Run(paramsMap)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		// render output according to chosen format
		text, err := renderResult(result)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		// write to file or stdout
		if output != "" {
			if err := os.WriteFile(output, text, 0o644); err != nil {
				cmd.PrintErrln(err)
			}
			return
		}

		fmt.Print(string(text))
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&format, "format", "f", formatYAML, "Output format (yaml, json or template)")
	runCmd.Flags().StringVarP(&output, "output", "o", "", "Output file")
	runCmd.Flags().StringVarP(&templatePath, "template", "t", "", "Go template file (used with -f template)")
	runCmd.Flags().StringSliceVarP(&params, "param", "p", nil, "InputSchema to pass to the template (used with -f template) in key=value format")
}

// validateRunArgs checks basic flag constraints and file existence.
func validateRunArgs(args []string) error {
	if format == formatTemplate && templatePath == "" {
		return fmt.Errorf("template path must be specified when using format=template")
	}
	if _, err := os.Stat(args[0]); err != nil {
		return fmt.Errorf("input file error: %w", err)
	}
	if format != formatYAML && format != formatJSON && format != formatTemplate {
		return fmt.Errorf("unsupported format %q", format)
	}
	return nil
}

// renderResult serializes the runner result according to the chosen format.
func renderResult(out any) ([]byte, error) {
	switch format {
	case formatJSON:
		return json.MarshalIndent(out, "", " ")
	case formatTemplate:
		if _, err := os.Stat(templatePath); err != nil {
			return nil, err
		}
		tplText, err := os.ReadFile(templatePath)
		if err != nil {
			return nil, err
		}
		tpl, err := template.New("template").Parse(string(tplText))
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := tpl.Execute(&buf, out); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	default: // yaml
		return yaml.Marshal(out)
	}
}

// newGeminiClient constructs a genai client using the configured service account.
func newGeminiClient() (*genai.Client, error) {
	credsBytes, err := os.ReadFile(cfg.GeminiServiceAccountPath)
	if err != nil {
		return nil, err
	}
	creds, err := credentials.DetectDefault(&credentials.DetectOptions{
		Scopes:          []string{"https://www.googleapis.com/auth/cloud-platform"},
		CredentialsJSON: credsBytes,
	})
	if err != nil {
		return nil, err
	}
	return genai.NewClient(context.Background(), &genai.ClientConfig{
		Project:     cfg.GeminiProjectID,
		Location:    cfg.GeminiLocation,
		Credentials: creds,
		Backend:     genai.BackendVertexAI,
	})
}

func sliceToMap(s []string) (map[string]string, error) {
	m := make(map[string]string, len(s))
	for _, v := range s {
		if matched, _ := regexp.Match("^[^=]+=[^=]+$", []byte(v)); !matched {
			return m, errors.New("invalid parameter format: " + v)
		}
		kv := strings.SplitN(v, "=", 2)
		m[kv[0]] = kv[1]
	}
	return m, nil
}
