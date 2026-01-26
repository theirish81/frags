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
package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMediaType(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "PDF file",
			filename: "document.pdf",
			expected: MediaPDF,
		},
		{
			name:     "Text file",
			filename: "file.txt",
			expected: MediaText,
		},
		{
			name:     "Markdown file",
			filename: "readme.md",
			expected: MediaMarkdown,
		},
		{
			name:     "CSV file",
			filename: "data.csv",
			expected: MediaCsv,
		},
		{
			name:     "Unsupported extension",
			filename: "image.jpg",
			expected: MediaText,
		},
		{
			name:     "No extension",
			filename: "file",
			expected: MediaText,
		},
		{
			name:     "Empty filename",
			filename: "",
			expected: MediaText,
		},
		{
			name:     "Extension only",
			filename: ".pdf",
			expected: MediaPDF,
		},
		{
			name:     "Uppercase extension PDF",
			filename: "test.PDF",
			expected: MediaPDF,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := GetMediaType(tc.filename)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
