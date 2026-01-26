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
	"github.com/samber/lo"
	"github.com/theirish81/frags/evaluators"
	"github.com/theirish81/frags/util"
)

type FunctionCallDestination string

const (
	AiFunctionCallDestination      FunctionCallDestination = "ai"
	VarsFunctionCallDestination    FunctionCallDestination = "vars"
	ContextFunctionCallDestination FunctionCallDestination = "context"
)

// FunctionCaller Represents a function invocation. NOTE: description is meant to explain to LLM what the output data is
// about when the function is called by an entity that's not the LLM itself.
// IMPORTANT: if Code is not nil, this will trigger the execution of the scripting engine. If the engine is nil, nothing
// will happen.
type FunctionCaller struct {
	Name        string                            `yaml:"name" json:"name"`
	Code        *string                           `yaml:"code" json:"code"`
	Args        map[string]any                    `yaml:"args" json:"args"`
	Description *string                           `yaml:"description" json:"description"`
	In          *FunctionCallDestination          `yaml:"in" json:"in" validate:"omitempty,oneof=ai vars context"`
	Var         *string                           `yaml:"var" json:"var"`
	Func        func(map[string]any) (any, error) `yaml:"-" json:"-"`
}

type FunctionCallers []FunctionCaller

func (f FunctionCallers) FilterVarsFunctionCalls() FunctionCallers {
	if f == nil {
		return FunctionCallers{}
	}
	fc := lo.Filter(f, func(fc FunctionCaller, index int) bool {
		return fc.In != nil && *fc.In == VarsFunctionCallDestination
	})
	return fc
}

func (f FunctionCallers) FilterAiFunctionCalls() FunctionCallers {
	fc := lo.Filter(f, func(fc FunctionCaller, index int) bool {
		return fc.In == nil || *fc.In == AiFunctionCallDestination
	})
	return fc
}

func (f FunctionCallers) FilterContextFunctionCalls() FunctionCallers {
	fc := lo.Filter(f, func(fc FunctionCaller, index int) bool {
		return fc.In != nil && *fc.In == ContextFunctionCallDestination
	})
	return fc
}

// RunAllFunctionCallers runs all the function calls in the given collection.
func (r *Runner[T]) RunAllFunctionCallers(ctx *util.FragsContext, fc FunctionCallers, scope evaluators.EvalScope) (map[string]any, error) {
	vx := make(map[string]any)
	for _, c := range fc {
		if ctx.Err() != nil {
			return vx, ctx.Err()
		}
		varName := c.Name + "_" + uuid.NewString()
		if c.Var != nil {
			varName = *c.Var
		}
		var err error
		vx[varName], err = r.runFunctionCaller(ctx, c, scope)
		scope.WithVars(vx)
		if err != nil {
			return vx, err
		}
	}
	return vx, nil
}

// runFunctionCaller runs a FunctionCaller object, evaluating the arguments if needed.
func (r *Runner[T]) runFunctionCaller(ctx *util.FragsContext, fc FunctionCaller, scope evaluators.EvalScope) (any, error) {
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
		return fc.Func(clonedFc.Args)
	} else if fc.Code != nil {
		return r.ScriptEngine().RunCode(ctx, *clonedFc.Code, clonedFc.Args, r)
	} else {
		return r.ai.RunFunction(ctx, clonedFc, r)
	}
}

// RunSessionAiPreCallsToTextContext runs the pre-call functions and composes a textual context to be prepended to the
// actual prompt.
func (r *Runner[T]) RunSessionAiPreCallsToTextContext(ctx *util.FragsContext, session Session, scope evaluators.EvalScope) (string, error) {
	preCallsText := ""
	if session.PreCalls != nil {
		for _, c := range session.PreCalls.FilterAiFunctionCalls() {
			res, err := r.runFunctionCaller(ctx, c, scope)
			if err != nil {
				return preCallsText, err
			}
			preCallsText += preCallCtx(c, res)
		}
	}
	return preCallsText, nil
}
