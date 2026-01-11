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
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type T struct {
	P1 string `json:"p1"`
	P2 string `json:"p2"`
	P3 string `json:"p3"`
	P4 string `json:"p4"`
}

func TestRunner_Run(t *testing.T) {
	sessionData, _ := os.ReadFile("test_data/sessions.yaml")
	mgr := NewSessionManager()
	err := mgr.FromYAML(sessionData)
	assert.Nil(t, err)
	ai := NewDummyAi()
	runner := NewRunner[T](mgr, NewDummyResourceLoader(), ai)
	out, err := runner.Run(map[string]string{"animal": "dog", "animals": "giraffes"})

	assert.Nil(t, err)
	assert.NotEmpty(t, out.P1)
	assert.NotEmpty(t, out.P2)
	assert.NotEmpty(t, out.P3)
	assert.NotEmpty(t, out.P4)

	assert.Equal(t, "extract these images data. Make sure they contain dog", out.P3)
	assert.Equal(t, "also these giraffes", out.P4)

}

func TestRunner_RunDependenciesAndContext(t *testing.T) {
	sessionData, _ := os.ReadFile("test_data/dependant_sessions.yaml")
	mgr := NewSessionManager()
	err := mgr.FromYAML(sessionData)
	assert.Nil(t, err)
	ai := NewDummyAi()
	runner := NewRunner[map[string]string](mgr, NewDummyResourceLoader(), ai, WithSessionWorkers(3))
	out, err := runner.Run(nil)
	assert.Nil(t, err)
	assert.Contains(t, (*out)["summary"], "CURRENT CONTEXT")
	assert.Contains(t, (*out)["summary"], "animal1")
	_, ok := (*out)["nop"]
	assert.False(t, ok)
}

func TestRunner_LoadSessionResource(t *testing.T) {
	sessionData, _ := os.ReadFile("test_data/session_resources.yaml")
	mgr := NewSessionManager()
	err := mgr.FromYAML(sessionData)
	assert.Nil(t, err)
	ai := NewDummyAi()
	runner := NewRunner[map[string]string](mgr, NewFileResourceLoader("./test_data"), ai, WithSessionWorkers(3))
	runner.dataStructure = &map[string]string{}
	res, err := runner.loadSessionResources(mgr.Sessions["s1"])
	assert.NoError(t, err)
	assert.Equal(t, "stuff.csv", res[0].Identifier)
	assert.Equal(t, MediaJson, res[0].MediaType)
	fmt.Println(string(res[0].ByteContent))
	out := make([]any, 0)
	err = json.Unmarshal(res[0].ByteContent, &out)
	assert.Nil(t, err)
	assert.Equal(t, []any{
		map[string]any{"first_name": "john", "last_name": "doe"},
		map[string]any{"first_name": "bill", "last_name": "murray"},
	}, out)
}
