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
	"slices"

	"github.com/theirish81/frags/evaluators"
)

// Dependency defines whether this session can run or should:
// * wait on another Session to complete
// * run at all, based on an Expression
type Dependency struct {
	Session    *string `json:"session" yaml:"session"`
	Expression *string `json:"expression" yaml:"expression"`
}

// Dependencies is a list of Dependencies
type Dependencies []Dependency

// DependencyCheckResult is the result of a dependency check.
type DependencyCheckResult string

const (
	DependencyCheckPassed     DependencyCheckResult = "passed"
	DependencyCheckFailed     DependencyCheckResult = "failed"
	DependencyCheckUnsolvable DependencyCheckResult = "unsolvable"
)

// CheckDependencies checks whether a session can start, cannot start yet, or will never start
func (r *Runner[T]) CheckDependencies(dependencies Dependencies) (DependencyCheckResult, error) {
	if dependencies == nil {
		return DependencyCheckPassed, nil
	}
	for _, dep := range dependencies {
		if dep.Session != nil {
			dependencyStatus, _ := r.status.Load(*dep.Session)
			if slices.Contains([]SessionStatus{failedSessionStatus, noOpSessionStatus}, dependencyStatus) {
				return DependencyCheckUnsolvable, nil
			}
			if slices.Contains([]SessionStatus{queuedSessionStatus, committedSessionStatus, runningSessionStatus}, dependencyStatus) {
				return DependencyCheckFailed, nil
			}
		}

		if dep.Expression != nil {
			pass, err := evaluators.EvaluateBooleanExpression(*dep.Expression, r.newEvalScope().WithVars(r.vars))
			if err != nil {
				return DependencyCheckUnsolvable, err
			}
			if !pass {
				return DependencyCheckUnsolvable, nil
			}
		}
	}
	return DependencyCheckPassed, nil
}
