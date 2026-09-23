package utils

import (
	"net/mail"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const (
	minDomainLength = 4
	maxDomainLength = 253
	maxEmailLength  = 254
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

func NormalizeEmail(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))

	if value == "" || len(value) > maxEmailLength {
		return ""
	}
	if strings.ContainsAny(value, " \t\r\n<>,;\"") {
		return ""
	}

	at := strings.LastIndex(value, "@")
	if at <= 0 || at == len(value)-1 {
		return ""
	}

	if _, err := mail.ParseAddress(value); err != nil {
		return ""
	}
	if !ValidDomain(value[at+1:]) {
		return ""
	}

	return value
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

func ParseIDList(raw string) []int {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	ids := make([]int, 0, len(parts))

	for _, part := range parts {
		parsed, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || parsed < 1 {
			continue
		}

		ids = append(ids, parsed)
	}

	return NormalizeIDs(ids)
}

func ParseEnumList(raw string, allowed ...string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	permitted := make(map[string]bool, len(allowed))
	for _, value := range allowed {
		permitted[value] = true
	}

	seen := make(map[string]bool, len(allowed))
	values := make([]string, 0, len(allowed))

	for _, part := range strings.Split(raw, ",") {
		value := strings.ToUpper(strings.TrimSpace(part))
		if !permitted[value] || seen[value] {
			continue
		}

		seen[value] = true
		values = append(values, value)
	}

	return values
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
