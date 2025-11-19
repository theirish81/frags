package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"cloud.google.com/go/auth/credentials"
	"github.com/spf13/cobra"
	"github.com/theirish81/frags"
	"github.com/theirish81/frags/gemini"
	"google.golang.org/genai"
	"gopkg.in/yaml.v3"
)

var format string
var output string
var templatePath string

var rootCmd = cobra.Command{
	Use: filepath.Base(os.Args[0]),
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a session",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if format == "template" && len(templatePath) == 0 {
			fmt.Println("template path file must be specified")
			return
		}
		if _, err := os.Stat(args[0]); err != nil {
			fmt.Println(err.Error())
			return
		}
		data, err := os.ReadFile(args[0])
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		sm := frags.NewSessionManager()
		if err := sm.FromYAML(data); err != nil {
			fmt.Println(err.Error())
			return
		}
		client, err := newGeminiClient()
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		dir := filepath.Dir(args[0])
		workers := cfg.ParallelWorkers
		if workers == 0 {
			workers = 1
		}
		runner := frags.NewRunner[frags.ProgMap](sm, frags.NewFileResourceLoader(dir), gemini.NewAI(client), frags.WithSessionWorkers(workers))
		out, err := runner.Run()
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		text := make([]byte, 0)
		switch format {
		case "json":
			text, err = json.MarshalIndent(out, "", " ")
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		case "template":
			if _, err := os.Stat(templatePath); err != nil {
				fmt.Println(err.Error())
				return
			}
			templateText, err := os.ReadFile(templatePath)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			tpl, err := template.New("template").Parse(string(templateText))
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			writer := bytes.NewBuffer(text)
			if err := tpl.Execute(writer, out); err != nil {
				fmt.Println(err.Error())
				return
			}
			text = writer.Bytes()

		default:
			text, err = yaml.Marshal(out)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		}
		if output != "" {
			err = os.WriteFile(output, text, 0644)
			if err != nil {
				fmt.Println(err.Error())
			}
		} else {
			fmt.Println(string(text))
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&format, "format", "f", "yaml", "Output format (yaml or json)")
	runCmd.Flags().StringVarP(&output, "output", "o", "", "Output file")
	runCmd.Flags().StringVarP(&templatePath, "template", "t", "", "Template file")
}

func newGeminiClient() (*genai.Client, error) {
	credsBytes, err := os.ReadFile(cfg.GeminiServiceAccountPath)
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
