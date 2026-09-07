package utils

import (
	"regexp"
	"slices"
	"strings"
)

const (
	minDomainLength = 4
	maxDomainLength = 253
)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

var domainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*\.[a-z]{2,63}$`)

func NormalizeDomain(domain string) string {
	value := strings.ToLower(strings.TrimSpace(domain))
	value = strings.TrimPrefix(value, "https://")
	value = strings.TrimPrefix(value, "http://")
	value = strings.TrimPrefix(value, "www.")
	value = strings.TrimSuffix(value, ".")

	return value
}

func ValidDomain(domain string) bool {
	if len(domain) < minDomainLength || len(domain) > maxDomainLength {
		return false
	}

	return domainPattern.MatchString(domain)
}

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
