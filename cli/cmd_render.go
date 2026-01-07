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
	"fmt"
	"os"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/theirish81/frags"
	"gopkg.in/yaml.v3"
)

var renderCmd = &cobra.Command{
	Use:   "render <path/to/data.json>",
	Short: "render a YAML/JSON data file into a document using a template",
	Long: `
Render a YAML/JSON data file into a document using a template. To be used in case your prefer generating the data output
and only later make the output into a document, which is particularly useful during the design phase of the template.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, err := os.ReadFile(args[0])
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		if templatePath == "" {
			cmd.PrintErrln("template path must be specified")
			return
		}
		format = formatTemplate
		progMap := frags.ProgMap{}
		if err := yaml.Unmarshal(data, &progMap); err != nil {
			cmd.PrintErrln(err)
			return
		}
		text, err := renderResult(progMap)
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
	renderCmd.Flags().StringVarP(&templatePath, "template", "t", "", "Go template file")
	renderCmd.Flags().StringVarP(&output, "output", "o", "", "Output file")
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
