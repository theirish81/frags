/*
 * Copyright (C) 2026 Simone Pezzano
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

/*
 * Copyright (C) 2026 Simone Pezzano
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

package frags

import (
	"github.com/diaphora-ai/zealql"
	"github.com/theirish81/frags/schema"
	"github.com/theirish81/frags/util"
)

var internalDbSystemPrompt = `
Internal database Notes:
* Use it to store arrays of objects
* Query it when the data is not available in the context. Don't check it if the expected result is not an array
* Produce queries for the SQL-Lite dialect that will most definitely work 
`

type InternalTools struct {
	db              *zealql.Database
	scriptingEngine ScriptEngine
}

func NewInternalTools(runner ExportableRunner) *InternalTools {
	return &InternalTools{db: runner.DB(), scriptingEngine: runner.ScriptEngine()}
}

func (c InternalTools) Name() string {
	return "internal_db_functions"
}

func (c InternalTools) Description() string {
	return "functions to interact with the internal database"
}

func (c InternalTools) AsToolDefinitions() ToolDefinitions {
	def := ToolDefinitions{}
	for _, f := range c.AsFunctions() {
		def = append(def, ToolDefinition{
			Name:        f.Name,
			Type:        ToolTypeFunction,
			InputSchema: f.Schema,
			Description: f.Description,
		})
	}
	return def
}

func (c InternalTools) AsFunctions() ExternalFunctions {
	return ExternalFunctions{
		"list_internal_db_tables": {
			Name:        "list_internal_db_tables",
			Description: "lists all the tables in the internal database.",
			Collection:  "internal_db_functions",
			Schema: &schema.Schema{
				Type: schema.Object,
			},
			Func: func(ctx *util.FragsContext, data map[string]any) (any, error) {
				return c.db.ListTables(), nil
			},
		},
		"describe_internal_db_tables": {
			Name:        "describe_internal_db_tables",
			Description: "describes multiple tables in the internal database",
			Collection:  "internal_db_functions",
			Func: func(ctx *util.FragsContext, data map[string]any) (any, error) {
				descriptions := make([]string, 0)
				for _, tn := range data["table_names"].([]any) {
					if table, ok := c.db.GetTable(tn.(string)); ok {
						sql := table.ToSQL()
						descriptions = append(descriptions, sql)
					}
				}
				return descriptions, nil
			},
			Schema: &schema.Schema{
				Type:        schema.Object,
				Required:    []string{"table_names"},
				Description: "the tables to describe",
				Properties: map[string]*schema.Schema{
					"table_names": {
						Type: schema.Array,
						Items: &schema.Schema{
							Type: schema.String,
						},
					},
				},
			},
		},
		"query_internal_db_tables": {
			Name:        "query_internal_db_tables",
			Description: "queries the internal database",
			Collection:  "internal_db_functions",
			Func: func(ctx *util.FragsContext, data map[string]any) (any, error) {
				return c.db.Query(data["query"].(string))
			},
			Schema: &schema.Schema{
				Type:        schema.Object,
				Required:    []string{"query"},
				Description: "the SQL-Lite compatible query",
				Properties: map[string]*schema.Schema{
					"query": {
						Type: schema.String,
					},
				},
			},
		},
		"insert_in_internal_db_table": {
			Name:        "insert_in_internal_db_table",
			Description: "insert data into the database. If the table does not exist, it will be created upon insertion.",
			Func: func(ctx *util.FragsContext, data map[string]any) (any, error) {
				table, err := c.db.CreateTable(data["table_name"].(string), data["records"].([]any))
				if err != nil {
					return nil, err
				}
				return table.ToSQL(), nil
			},
			Schema: &schema.Schema{
				Type:        schema.Object,
				Required:    []string{"table_name", "records"},
				Description: "the table to insert into",
				Properties: map[string]*schema.Schema{
					"table_name": {
						Type: schema.String,
					},
					"records": {
						Type: schema.Array,
						Items: &schema.Schema{
							Type: schema.Object,
						},
					},
				},
			},
		},
		"execute_javascript": {
			Name:        "execute_javascript",
			Description: "execute JavaScript code (using completion-value notation) for the sole purpose of number crunching and data reshaping. No NodeJS objects are allowed (console.log... etc)",
			Func: func(ctx *util.FragsContext, data map[string]any) (any, error) {
				return c.scriptingEngine.RunCode(ctx, data["code"].(string), data["args"], nil)
			},
			Schema: &schema.Schema{
				Type:     schema.Object,
				Required: []string{"code", "args"},
				Properties: map[string]*schema.Schema{
					"code": {
						Type:        schema.String,
						Description: "the JavaScript code to execute",
						Example:     "var t = args.raw.split(',').map(t => t.trim())\nt;",
					},
					"args": {
						Type:        schema.Object,
						Description: "the arguments to pass to the code. They will be exposed to the engine as the object `args`. Do not inline the arguments in the code",
					},
				},
			},
		},
	}
}
