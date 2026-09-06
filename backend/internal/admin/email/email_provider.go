package email

import (
	"regexp"
	"strings"
)

const (
	ProviderOutlook365 = "outlook365"
	ProviderGmail      = "gmail"
)

const (
	minDomainLength = 4
	maxDomainLength = 253
	defaultPage     = 1
	defaultPageSize = 25
	maxPageSize     = 100
)

var domainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*\.[a-z]{2,63}$`)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func NormalizeName(name string) string {
	return strings.Join(strings.Fields(name), " ")
}

func NormalizeDomain(domain string) string {
	value := strings.ToLower(strings.TrimSpace(domain))
	value = strings.TrimPrefix(value, "https://")
	value = strings.TrimPrefix(value, "http://")
	value = strings.TrimPrefix(value, "www.")
	value = strings.TrimSuffix(value, ".")

	return value
}

func NormalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func ValidDomain(domain string) bool {
	if len(domain) < minDomainLength || len(domain) > maxDomainLength {
		return false
	}

	return domainPattern.MatchString(domain)
}

func ValidProvider(provider string) bool {
	return provider == ProviderOutlook365 || provider == ProviderGmail
}

func normalizePaging(page, pageSize int) (int, int) {
	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return page, pageSize
}

func escapeLike(value string) string {
	return likeEscaper.Replace(value)
}
