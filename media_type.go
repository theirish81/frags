package frags

import "path/filepath"

const ExtensionPDF = ".pdf"
const ExtensionTXT = ".txt"
const ExtensionMD = ".md"
const ExtensionCSV = ".csv"

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
