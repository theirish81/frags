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

import "path/filepath"

const ExtensionPDF = ".pdf"
const ExtensionTXT = ".txt"
const ExtensionMD = ".md"
const ExtensionCSV = ".csv"

// NOTE: while markdown and csv technically have their own content types, we use text/plain because the LLM either
// doesn't care, or even likes it better.

const MediaPDF = "application/pdf"
const MediaText = "text/plain"
const MediaMarkdown = "text/plain"
const MediaCsv = "text/plain"

// GetMediaType returns the media type for a given file extension
func GetMediaType(filename string) string {
	switch filepath.Ext(filename) {
	case ExtensionPDF:
		return MediaPDF
	case ExtensionTXT:
		return MediaText
	case ExtensionMD:
		return MediaMarkdown
	case ExtensionCSV:
		return MediaCsv
	}
	return MediaText
}
