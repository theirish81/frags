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

// ScriptEngine is the interface that wraps the RunCode method. Frags provides NO script engines, it's the program
// that includes Frags that provides one, if necessary. Beware though, most script engines pose a security risk.
type ScriptEngine interface {
	RunCode(code string, params map[string]any, runner ExportableRunner) (map[string]any, error)
}

type DummyScriptEngine struct{}

func (d *DummyScriptEngine) RunCode(_ string, _ map[string]any, _ ExportableRunner) (map[string]any, error) {
	return make(map[string]any), nil
}
