package relay

import (
	"context"
	"errors"
	"net"
	"sort"
	"strings"
	"time"
)

var ErrNoDestination = errors.New("no mail exchanger or address record for domain")

type Destination struct {
	Host string
	Addr string
}

type Resolver struct {
	resolver *net.Resolver
	timeout  time.Duration
	ipv6     bool
	override string
}

func NewResolver(timeout time.Duration, ipv6 bool, override string) *Resolver {
	return &Resolver{resolver: net.DefaultResolver, timeout: timeout, ipv6: ipv6, override: override}
}

func (r *Resolver) Destinations(ctx context.Context, domain string) ([]Destination, error) {
	if r.override != "" {
		host, port := splitOverride(r.override)

		return []Destination{{Host: host, Addr: net.JoinHostPort(host, port)}}, nil
	}

	lookupCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	hosts, err := r.hosts(lookupCtx, domain)
	if err != nil {
		return nil, err
	}

	destinations := make([]Destination, 0, len(hosts))

	for _, host := range hosts {
		addresses, err := r.addresses(lookupCtx, host)
		if err != nil {
			continue
		}

		for _, address := range addresses {
			destinations = append(destinations, Destination{Host: host, Addr: net.JoinHostPort(address, "25")})
		}
	}

	if len(destinations) == 0 {
		return nil, ErrNoDestination
	}

	return destinations, nil
}

func (r *Resolver) hosts(ctx context.Context, domain string) ([]string, error) {
	records, err := r.resolver.LookupMX(ctx, domain)
	if err == nil && len(records) > 0 {
		sort.SliceStable(records, func(i, j int) bool { return records[i].Pref < records[j].Pref })

		hosts := make([]string, 0, len(records))
		for _, record := range records {
			hosts = append(hosts, strings.TrimSuffix(record.Host, "."))
		}

		return hosts, nil
	}

	if err != nil && !temporary(err) {
		if _, lookupErr := r.resolver.LookupHost(ctx, domain); lookupErr != nil {
			return nil, ErrNoDestination
		}
	}

	return []string{domain}, nil
}

func (r *Resolver) addresses(ctx context.Context, host string) ([]string, error) {
	ips, err := r.resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}

	addresses := make([]string, 0, len(ips))

	for _, ip := range ips {
		if ip.IP.To4() == nil && !r.ipv6 {
			continue
		}

		addresses = append(addresses, ip.IP.String())
	}

	if len(addresses) == 0 {
		return nil, ErrNoDestination
	}

	return addresses, nil
}

func splitOverride(override string) (string, string) {
	host, port, err := net.SplitHostPort(override)
	if err != nil {
		return override, "25"
	}

	return host, port
}

func temporary(err error) bool {
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return dnsErr.IsTemporary || dnsErr.IsTimeout
	}

	return false
}
