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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateArrayExpression(t *testing.T) {
	t.Run("successfully evaluates an array expression", func(t *testing.T) {
		res, err := EvaluateArrayExpression(`bar`, EvalScope{
			"bar": []string{"foo"},
		})
		assert.NoError(t, err)
		assert.Equal(t, "foo", res[0])

	})
	t.Run("returns an error if the data does not resolve to an array", func(t *testing.T) {
		_, err := EvaluateArrayExpression(`bar`, EvalScope{
			"bar": 123,
		})
		assert.Error(t, err)
	})

	t.Run("returns correct data type for the selected array", func(t *testing.T) {
		res, err := EvaluateArrayExpression(`val`, EvalScope{
			"val": []s1{
				{
					S2: s2{
						P1: 123,
					},
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, float64(123), res[0].(s1).S2.P1)
	})
}

func TestEvaluateBooleanExpression(t *testing.T) {
	t.Run("evaluates a true boolean expression", func(t *testing.T) {
		res, err := EvaluateBooleanExpression(`"foo" in bar`, EvalScope{
			"bar": []string{"foo"},
		})
		assert.NoError(t, err)
		assert.True(t, res)
	})

	t.Run("evaluates a false boolean expression", func(t *testing.T) {
		res, err := EvaluateBooleanExpression(`"fuz" in bar`, EvalScope{
			"bar": []string{"foo"},
		})
		assert.NoError(t, err)
		assert.False(t, res)
	})

	t.Run("fails to evaluate boolean", func(t *testing.T) {
		res, err := EvaluateBooleanExpression(`bar`, EvalScope{
			"bar": []string{"foo"},
		})
		assert.Error(t, err)
		assert.False(t, res)
	})

}

func TestEvaluateTemplate(t *testing.T) {
	res, err := EvaluateTemplate(`{{.foo}}`, EvalScope{
		"foo": "bar",
	})
	assert.Nil(t, err)
	assert.Equal(t, "bar", res)

	res, err = EvaluateTemplate(`{{.foo}}`, EvalScope{
		"foo":  "{{.buzz}}",
		"buzz": "bar",
	})
	assert.Nil(t, err)
	assert.Equal(t, "bar", res)
}

func TestEvalScope_WithVars(t *testing.T) {
	scope := NewEvalScope()
	scope = scope.WithVars(map[string]any{
		"buzz": "baz",
	})
	assert.Equal(t, "baz", scope.Vars()["buzz"])

	scope = scope.WithVars(map[string]any{
		"hey": "joe",
	})
	assert.Equal(t, "baz", scope.Vars()["buzz"])
	assert.Equal(t, "joe", scope.Vars()["hey"])

	scope = scope.WithVars(map[string]any{
		"hey": "mario",
		"red": "ball",
	})
	assert.Equal(t, "baz", scope.Vars()["buzz"])
	assert.Equal(t, "mario", scope.Vars()["hey"])
	assert.Equal(t, "ball", scope.Vars()["red"])
}

func TestEvaluateExprFunctions(t *testing.T) {
	data := []any{map[string]any{"foo": "bar"}, map[string]any{"foo": "bar"}, map[string]any{"foo": "baz"}}
	arr, err := EvaluateArrayExpression("unique(map(data, .foo))", EvalScope{"data": data})
	assert.NoError(t, err)
	fmt.Println(arr)
}
