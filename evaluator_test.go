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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateArrayExpression(t *testing.T) {
	res, err := EvaluateArrayExpression(`bar`, EvalScope{
		"bar": []string{"foo"},
	})
	assert.Nil(t, err)
	assert.Equal(t, "foo", res[0])

	res, err = EvaluateArrayExpression(`bar`, EvalScope{
		"bar": 123,
	})
	assert.NotNil(t, err)

	res, err = EvaluateArrayExpression(`val`, EvalScope{
		"val": []s1{
			{
				S2: s2{
					P1: 123,
				},
			},
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, float64(123), res[0].(s1).S2.P1)
}

func TestEvaluateBooleanExpression(t *testing.T) {
	res, err := EvaluateBooleanExpression(`"foo" in bar`, EvalScope{
		"bar": []string{"foo"},
	})
	assert.Nil(t, err)
	assert.True(t, res)

	res, err = EvaluateBooleanExpression(`"fuz" in bar`, EvalScope{
		"bar": []string{"foo"},
	})
	assert.Nil(t, err)
	assert.False(t, res)
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
