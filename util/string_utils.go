package util

import "strings"

func Split(identifiers []string, limit int) [][]string {
	if limit < 0 {
		limit = -1 * limit
	} else if limit == 0 {
		return [][]string{identifiers}
	}

	var chunk []string
	chunks := make([][]string, 0, len(identifiers)/limit+1)
	for len(identifiers) >= limit {
		chunk, identifiers = identifiers[:limit], identifiers[limit:]
		chunks = append(chunks, chunk)
	}
	if len(identifiers) > 0 {
		chunks = append(chunks, identifiers[:])
	}

	return chunks
}

// Difference returns the elements in `a` that aren't in `b`.
func Difference(a, b []*string) []*string {
	mb := make(map[string]bool, len(b))
	for _, x := range b {
		mb[*x] = true
	}

	var diff []*string
	for _, x := range a {
		if _, found := mb[*x]; !found {
			diff = append(diff, x)
		}
	}

	return diff
}

// Truncate accepts a string and a max length. If the max length is less than the string's current length,
// then only the first maxLen characters of the string are returned, with "..." appended.
func Truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// RemoveNewlines will delete all the newlines (\n and \r) in a given string, which is useful for making
// error messages "sit" more nicely within their specified table cells in the terminal
func RemoveNewlines(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", " ")
}

// ToStringPtrSlice converts a slice of strings to a slice of string pointers.
func ToStringPtrSlice(strs []string) []*string {
	result := make([]*string, len(strs))
	for i := range strs {
		result[i] = &strs[i]
	}
	return result
}
