package frags

import "path/filepath"

const ExtensionPDF = ".pdf"
const ExtensionTXT = ".txt"
const ExtensionMD = ".md"

const MediaPDF = "application/pdf"
const MediaText = "text/plain"
const MediaMarkdown = "text/plain"

func GetMediaType(filename string) string {
	switch filepath.Ext(filename) {
	case ExtensionPDF:
		return MediaPDF
	case ExtensionTXT:
		return MediaText
	case ExtensionMD:
		return MediaMarkdown
	}
	return MediaText
}
