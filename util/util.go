package util

// BoolToInt converts bool to int.
func BoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
