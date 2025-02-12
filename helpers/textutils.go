// helpers/textutils.go
package helpers

// TrimText trims the text to 45 characters if it's longer
func TrimText(text string) string {
	const maxLength = 45
	if len(text) > maxLength {
		return text[:maxLength]
	}
	return text
}

// PadText pads the text with spaces until it's 45 characters long
func PadText(text string) string {
	const targetLength = 45
	if len(text) >= targetLength {
		return text
	}
	padding := targetLength - len(text)
	return text + string(make([]byte, padding))
}
