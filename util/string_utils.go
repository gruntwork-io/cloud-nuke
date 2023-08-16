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
