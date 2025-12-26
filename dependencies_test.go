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

func TestRunner_CheckDependencies(t *testing.T) {
	rx := Runner[map[string]any]{
		status: &SafeMap[string, SessionStatus]{
			data: map[string]SessionStatus{
				"foo": queuedSessionStatus,
				"fuz": finishedSessionStatus,
				"bar": runningSessionStatus,
				"baz": failedSessionStatus,
				"bat": noOpSessionStatus,
				"bam": committedSessionStatus,
			},
		},
		dataStructure: &map[string]any{
			"foo": "bar",
		},
	}
	res, err := rx.CheckDependencies(Dependencies{
		{
			Session: strPtr("foo"),
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, DependencyCheckFailed, res)

	res, err = rx.CheckDependencies(Dependencies{
		{
			Session: strPtr("fuz"),
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, DependencyCheckPassed, res)

	res, err = rx.CheckDependencies(Dependencies{
		{
			Session: strPtr("bar"),
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, DependencyCheckFailed, res)

	res, err = rx.CheckDependencies(Dependencies{
		{
			Session: strPtr("baz"),
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, DependencyCheckUnsolvable, res)

	res, err = rx.CheckDependencies(Dependencies{
		{
			Session: strPtr("bat"),
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, DependencyCheckUnsolvable, res)

	res, err = rx.CheckDependencies(Dependencies{
		{
			Session: strPtr("bam"),
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, DependencyCheckFailed, res)

	res, err = rx.CheckDependencies(Dependencies{
		{
			Session:    strPtr("fuz"),
			Expression: strPtr("context.foo == 'bar'"),
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, DependencyCheckPassed, res)

	res, err = rx.CheckDependencies(Dependencies{
		{
			Session:    strPtr("fuz"),
			Expression: strPtr("context.foo == 'ban'"),
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, DependencyCheckUnsolvable, res)

}
