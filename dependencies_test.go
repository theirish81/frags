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
