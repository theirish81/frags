/*
 * Copyright (C) 2025 Simone Pezzano
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	format       string
	output       string
	templatePath string
	params       []string
	debug        bool
	prePrompt    string
	systemPrompt string
	uploads      []string
)

var rootCmd = cobra.Command{
	Use:   filepath.Base(os.Args[0]),
	Short: "CLI for Frags. Run a Frags session from a YAML file.",
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(askCmd)
	rootCmd.AddCommand(renderCmd)

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
