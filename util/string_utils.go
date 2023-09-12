package util

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
