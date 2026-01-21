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

func TestToMarkdown(t *testing.T) {
	assert.Equal(t, "\n- **doh**: `(string)` \"val_doh\"\n- **foo**: `(string)` \"bar\"\n- **list**: \n  - [0] `(string)` \"a\"\n  - [1] `(string)` \"b\"\n  - [2] `(string)` \"c\"\n- **sub**: \n  - **baz**: `(string)` \"qux\"\n  - **p2**: \n    - **p3**: `(string)` \"val_p3\"", ToKFormat(ProgMap{
		"foo": "bar",
		"sub": ProgMap{
			"baz": "qux",
			"p2": map[string]string{
				"p3": "val_p3",
			},
		},
		"gee":  nil,
		"doh":  StrPtr("val_doh"),
		"list": []string{"a", "b", "c"},
	}))

	assert.Equal(t, "\n- **S2**: \n  - **P1**: `(float)` 123\n  - **P2**: `(float)` 0\n- **ArrayOfS2**:  (empty)\n- **MapOfS3**: (empty)",
		ToKFormat(s1{S2: s2{P1: 123}}))
}
