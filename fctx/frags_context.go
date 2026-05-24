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

package fctx

import (
	"context"
	"time"

	"github.com/theirish81/frags/evaluators"
	"github.com/theirish81/frags/util"
)

type FragsContext struct {
	context.Context
	scope        *evaluators.EvalScope
	ProgramError error
	cancel       context.CancelFunc
}

func NewFragsContext(timeout time.Duration) *FragsContext {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return &FragsContext{Context: ctx, cancel: cancel}
}

func WithFragsContext(ctx context.Context, timeout time.Duration) *FragsContext {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	return &FragsContext{Context: ctx, cancel: cancel}
}

func (f *FragsContext) Cancel(err error) {
	if err != nil {
		f.ProgramError = err
	}
	f.cancel()
}

func (f *FragsContext) Child(timeout time.Duration) *FragsContext {
	ctx, cancel := context.WithTimeout(f, timeout)
	return &FragsContext{Context: ctx, cancel: cancel}
}

func (f *FragsContext) Err() error {
	if f.Context.Err() == nil {
		return nil
	}
	if f.ProgramError == nil {
		return f.Context.Err()
	}
	return &util.CtxError{Err1: f.Context.Err(), Err2: f.ProgramError}
}

func (f *FragsContext) WithScope(evaluator *evaluators.EvalScope) *FragsContext {
	cx := *f
	cx.scope = evaluator
	return &cx
}

func (f *FragsContext) Scope() *evaluators.EvalScope {
	return f.scope
}
