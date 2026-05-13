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

package scoper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNode(t *testing.T) {
	node := Node("gino", "").Name("some content").ContentType("application/json").Description("some description")
	node.AppendChild(Node("child1", "some other content"))
	data := node.String()
	assert.Equal(t, "<gino name=\"some content\" description=\"some description\" contentType=\"application/json\">\n <child1><![CDATA[ some other content ]]></child1>\n</gino>", data)
}
