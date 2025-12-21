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

package functions

import (
	"os"

	"github.com/theirish81/frags"
)

var Fs frags.Functions = frags.Functions{
	"fs_list_files": {
		Description: "lists files in a provided directory",
		Schema: &frags.Schema{
			Type:     "object",
			Required: []string{"path"},
			Properties: map[string]*frags.Schema{
				"path": {Type: "string"},
			},
		},
		Func: func(args map[string]any) (map[string]any, error) {
			path, err := GetArg[string](args, "path")
			if err != nil {
				return nil, err
			}
			files, err := os.ReadDir(*path)
			if err != nil {
				return nil, err
			}
			return map[string]any{"files": files}, nil
		},
	},
	"fs_read_file": {
		Description: "reads a file and returns its contents",
		Schema: &frags.Schema{
			Type:     "object",
			Required: []string{"path"},
			Properties: map[string]*frags.Schema{
				"path": {Type: "string"},
			},
		},
		Func: func(args map[string]any) (map[string]any, error) {
			path, err := GetArg[string](args, "path")
			if err != nil {

			}
			contents, err := os.ReadFile(*path)
			if err != nil {
				return nil, err
			}
			return map[string]any{"contents": string(contents)}, nil
		},
	},
	"write_file": {
		Description: "writes a file with the provided contents",
		Schema: &frags.Schema{
			Type:     "object",
			Required: []string{"path", "content"},
			Properties: map[string]*frags.Schema{
				"path":    {Type: "string"},
				"content": {Type: "string"},
			},
		},
		Func: func(args map[string]any) (map[string]any, error) {
			path, err := GetArg[string](args, "path")
			if err != nil {
				return nil, err
			}
			content, err := GetArg[string](args, "content")
			if err != nil {
				return nil, err
			}
			return nil, os.WriteFile(*path, []byte(*content), 0644)
		},
	},
}
