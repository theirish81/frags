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

package frags

import (
	"github.com/google/uuid"
	"github.com/theirish81/frags/evaluators"
	"github.com/theirish81/frags/scoper"
	"github.com/theirish81/frags/util"
)

type FunctionCallDestination string

const (
	AiFunctionCallDestination      FunctionCallDestination = "ai"
	VarsFunctionCallDestination    FunctionCallDestination = "vars"
	ContextFunctionCallDestination FunctionCallDestination = "context"
	DbFunctionCallDestination      FunctionCallDestination = "db"
)

// FunctionCaller Represents a function invocation. NOTE: description is meant to explain to LLM what the output data is
// about when the function is called by an entity that's not the LLM itself.
// IMPORTANT: if Code is not nil, this will trigger the execution of the scripting engine. If the engine is nil, nothing
// will happen.
type FunctionCaller struct {
	Name        string                     `yaml:"name" json:"name"`
	Code        *string                    `yaml:"code" json:"code"`
	Args        map[string]any             `yaml:"args" json:"args"`
	Description *string                    `yaml:"description" json:"description"`
	In          *FunctionCallDestination   `yaml:"in" json:"in" validate:"omitempty,oneof=ai vars context"`
	Var         *string                    `yaml:"var" json:"var"`
	Func        FunctionCallerCallbackFunc `yaml:"-" json:"-"`
}

type FunctionCallerCallbackFunc func(ctx *util.FragsContext, data map[string]any) (any, error)

type FunctionCallers []FunctionCaller

// RunAllFunctionCallers runs all the function calls in the given collection.
func (r *Runner) RunAllFunctionCallers(ctx *util.FragsContext, fc FunctionCallers, inputScope evaluators.EvalScope, outputVars evaluators.Vars) (*scoper.KnowledgeNode, error) {
	vx := make(map[string]any)
	if len(fc) == 0 {
		return nil, nil
	}
	callsNode := scoper.Node("Calls", "")
	for _, c := range fc {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		varName := c.Name + "_" + uuid.NewString()
		if c.Var != nil {
			varName = *c.Var
		}
		var err error
		value, err := r.runFunctionCaller(ctx, c, inputScope)
		if err != nil {
			return nil, err
		}
		vx[varName] = value
		inputScope.WithVars(vx)

		if c.In == nil {
			c.In = util.Ptr(AiFunctionCallDestination)
		}
		switch *c.In {
		case VarsFunctionCallDestination:
			outputVars[varName] = value
		case ContextFunctionCallDestination:
			if err := util.SetInContext(r.dataStructure, varName, value); err != nil {
				return nil, err
			}
		case DbFunctionCallDestination:
			if r.db != nil {
				if _, err := r.db.Insert(varName, value); err != nil {
					return nil, err
				}
			}
		default:
			callsNode.AppendChild(preCallCtx(c, vx[varName]))
		}
	}
	return callsNode, nil
}

// runFunctionCaller runs a FunctionCaller object, evaluating the arguments if needed.
func (r *Runner) runFunctionCaller(ctx *util.FragsContext, fc FunctionCaller, scope evaluators.EvalScope) (any, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	clonedFc := fc
	var err error
	clonedFc.Args, err = evaluators.EvaluateMapValues(clonedFc.Args, scope)
	if err != nil {
		return nil, err
	}
	if fc.Func != nil {
		return fc.Func(ctx, clonedFc.Args)
	} else if fc.Code != nil {
		return r.ScriptEngine().RunCode(ctx, *clonedFc.Code, clonedFc.Args, r)
	} else {
		return r.RunFunction(ctx, clonedFc.Name, clonedFc.Args)
	}
}
