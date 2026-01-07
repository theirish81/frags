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
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
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
	Short: "the Frags agent as a CLI",
	Long: `
Frags is an advanced AI agent for complex data workflows—retrieval, transformation, extraction, and aggregation. Highly
customizable and extensible, it prioritizes precision.`,
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(askCmd)
	rootCmd.AddCommand(renderCmd)
	rootCmd.AddCommand(scriptCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(web)

}
