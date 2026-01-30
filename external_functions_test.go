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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/theirish81/frags/resources"
	"github.com/theirish81/frags/util"
)

func TestExternalFunction_Run(t *testing.T) {
	runner := NewRunner[util.ProgMap](NewSessionManager(), resources.NewDummyResourceLoader(), NewDummyAi())
	ef := ExternalFunction{
		Name: "f1",
		Func: func(ctx *util.FragsContext, args map[string]any) (any, error) {
			return args, nil
		},
	}
	runner.sessionManager.Transformers = &Transformers{
		{
			Name:            "t1",
			OnFunctionInput: util.StrPtr("f1"),
			Expr:            util.StrPtr("{\"yay\": args.foo}"),
		},
	}
	res, err := ef.Run(util.NewFragsContext(1*time.Minute), map[string]any{"foo": "bar"}, &runner)
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{"yay": "bar"}, res)
}
