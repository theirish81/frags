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
	"errors"

	"github.com/theirish81/frags/evaluators"
	"github.com/theirish81/frags/fctx"
	"github.com/theirish81/frags/schema"
)

type Interiorizer struct {
	scope *evaluators.EvalScope
}

func NewInteriorizer(scope *evaluators.EvalScope) *Interiorizer {
	return &Interiorizer{scope: scope}
}

func (i Interiorizer) AsFunctions() ExternalFunctions {
	return ExternalFunctions{
		"_describe_variable": ExternalFunction{
			Name:        "_describe_variable",
			Description: "describes a variable nature and shape",
			Schema: &schema.Schema{
				Type:     schema.Object,
				Required: []string{"variable_name"},
				Properties: map[string]*schema.Schema{
					"variable_name": {
						Type: schema.String,
					},
				},
			},
			Func: func(ctx *fctx.FragsContext, data map[string]any) (any, error) {
				varName := data["variable_name"].(string)
				if content, ok := i.scope.Vars()[varName]; ok {
					return schema.GuessSchema(content), nil
				}
				return nil, errors.New("variable not found")
			},
		},
		"_read_variable": ExternalFunction{
			Name:        "_read_variable",
			Description: "reads a variable content",
			Schema: &schema.Schema{
				Type:     schema.Object,
				Required: []string{"variable_name"},
				Properties: map[string]*schema.Schema{
					"variable_name": {
						Type: schema.String,
					},
				},
			},
		},
	}
}
