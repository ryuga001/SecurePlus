package utils

import (
	"slices"
	"strings"
)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func NormalizePaging(page, pageSize int) (int, int) {
	if page < 1 {
		page = DefaultPage
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	return page, pageSize
}

func Offset(page, pageSize int) int {
	return (page - 1) * pageSize
}

func EscapeLike(value string) string {
	return likeEscaper.Replace(value)
}

func LikePattern(search string) string {
	return "%" + EscapeLike(strings.TrimSpace(search)) + "%"
}

func NormalizeName(name string) string {
	return strings.Join(strings.Fields(name), " ")
}

func NormalizeIDs(ids []int) []int {
	cleaned := make([]int, 0, len(ids))

	for _, id := range ids {
		if id > 0 {
			cleaned = append(cleaned, id)
		}
	}

	slices.Sort(cleaned)

	return slices.Compact(cleaned)
}
