package nginx

import (
	"fmt"
	"path"
)

// Filter keeps the parsed log entries that match every configured allowlist and drops
// the rest, before the record ever reaches the Aggregator. Empty slices mean "no
// restriction" EXCEPT Methods: an empty Methods set is treated as GET-only because
// HEAD probes from podcast clients (Apple Podcasts, watchOS) land as 200 with 0 bytes
// and would otherwise inflate download counts.
type Filter struct {
	Methods map[string]struct{} // GET, HEAD, POST — upper-case; nil treated as GET-only
	Status  map[int]struct{}    // 200, 206, …; nil = any status allowed
	Paths   []string            // glob patterns matched against basename of path; nil = any path
	Hosts   map[string]struct{} // nginx $http_host; nil = any host
}

// NewFilter builds a Filter from CLI inputs. methods, statuses, paths and hosts may each
// be empty, in which case the field defaults as described on Filter. Each pattern in
// paths is validated eagerly so a bad glob fails startup, not every log line.
func NewFilter(methods []string, statuses []int, paths []string, hosts []string) (Filter, error) {
	f := Filter{}

	if len(methods) > 0 {
		f.Methods = make(map[string]struct{}, len(methods))
		for _, m := range methods {
			f.Methods[m] = struct{}{}
		}
	}

	if len(statuses) > 0 {
		f.Status = make(map[int]struct{}, len(statuses))
		for _, s := range statuses {
			f.Status[s] = struct{}{}
		}
	}

	for _, p := range paths {
		if _, err := path.Match(p, "test"); err != nil {
			return Filter{}, fmt.Errorf("invalid path glob %q: %w", p, err)
		}
	}
	f.Paths = paths

	if len(hosts) > 0 {
		f.Hosts = make(map[string]struct{}, len(hosts))
		for _, h := range hosts {
			f.Hosts[h] = struct{}{}
		}
	}

	return f, nil
}

// Keep reports whether a parsed line should be fed to the aggregator.
func (f Filter) Keep(p Parsed) bool {
	// Methods: empty set means GET-only (hard default, keeps HEAD out)
	if len(f.Methods) == 0 {
		if p.Method != "GET" {
			return false
		}
	} else if _, ok := f.Methods[p.Method]; !ok {
		return false
	}

	if len(f.Status) > 0 {
		if _, ok := f.Status[p.Status]; !ok {
			return false
		}
	}

	if len(f.Hosts) > 0 {
		if _, ok := f.Hosts[p.Host]; !ok {
			return false
		}
	}

	if len(f.Paths) > 0 {
		base := path.Base(p.Path)
		matched := false
		for _, pat := range f.Paths {
			if ok, _ := path.Match(pat, base); ok {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}
