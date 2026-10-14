// Package nginx parses access-log lines emitted by nginx with a fixed log_format
// extended for use with rlb-stats, and feeds them through a tailer + filter pipeline
// into store.LogRecord values the Aggregator understands.
package nginx

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/umputun/rlb-stats/app/store"
)

// LogFormat is the required nginx log_format directive expected by Parse. Any deployment
// pointing rlb-stats at an access log must configure nginx with this exact format so
// every field lands where the regex looks for it.
const LogFormat = `$remote_addr - - [$time_local] "$request" $status $body_bytes_sent ` +
	`"$http_host" "$http_referer" "$http_user_agent" $scheme ` +
	`"$http_cf_connecting_ip" "$http_cf_ipcountry" "$http_range"`

// timeLayout matches nginx's $time_local, e.g. 26/Apr/2026:21:52:30 +0000
const timeLayout = "02/Jan/2006:15:04:05 -0700"

// lineRE matches the full log_format. The three trailing CF fields are REQUIRED: a line
// without them fails parse so pre-update nginx output logs at DEBUG rather than being
// silently matched with empty CF values.
var lineRE = regexp.MustCompile(
	`^(?P<remote>\S+) \S+ \S+ \[(?P<time>[^\]]+)\] ` +
		`"(?P<request>[^"]*)" (?P<status>\d+) (?P<bytes>\d+) ` +
		`"(?P<host>[^"]*)" "(?P<referer>[^"]*)" "(?P<ua>[^"]*)" (?P<scheme>\S+) ` +
		`"(?P<cfip>[^"]*)" "(?P<cfcountry>[^"]*)" "(?P<range>[^"]*)"\s*$`,
)

// Parsed holds the raw fields a single log line yields; it is what Filter inspects before
// the record is converted to store.LogRecord. Keeping method, status, host and path out
// of LogRecord lets Filter decide without constructing the heavier record.
type Parsed struct {
	Method   string
	Status   int
	Path     string
	Host     string
	ClientIP string
	Country  string
	Bytes    int64
	Date     time.Time
}

// ErrBadLine indicates a line that does not match the required log_format. Callers log
// these at DEBUG; a nginx that still emits the pre-update format produces nothing but
// these until it is reconfigured.
var ErrBadLine = errors.New("line does not match expected log_format")

// Parse extracts the fields from one nginx log line. It returns ErrBadLine for any line
// that fails to match the full log_format, including old-format lines missing the three
// trailing CF fields.
func Parse(line string) (Parsed, error) {
	m := lineRE.FindStringSubmatch(line)
	if m == nil {
		return Parsed{}, fmt.Errorf("%w: %q", ErrBadLine, truncate(line, 200))
	}
	g := map[string]string{}
	for i, name := range lineRE.SubexpNames() {
		if name != "" {
			g[name] = m[i]
		}
	}

	t, err := time.Parse(timeLayout, g["time"])
	if err != nil {
		return Parsed{}, fmt.Errorf("parse time %q: %w", g["time"], err)
	}

	status, err := strconv.Atoi(g["status"])
	if err != nil {
		return Parsed{}, fmt.Errorf("parse status %q: %w", g["status"], err)
	}

	bytesVal, err := strconv.ParseInt(g["bytes"], 10, 64)
	if err != nil {
		return Parsed{}, fmt.Errorf("parse body_bytes_sent %q: %w", g["bytes"], err)
	}

	method, path, ok := splitRequest(g["request"])
	if !ok {
		return Parsed{}, fmt.Errorf("%w: malformed request %q", ErrBadLine, g["request"])
	}

	return Parsed{
		Method:   method,
		Status:   status,
		Path:     path,
		Host:     g["host"],
		ClientIP: resolveClientIP(g["cfip"], g["remote"]),
		Country:  resolveCountry(g["cfcountry"]),
		Bytes:    bytesVal,
		Date:     t,
	}, nil
}

// ToRecord builds the store.LogRecord that lands in the aggregator. DestHost is the Host
// header so each nginx vhost (radio-t.com, cdn.radio-t.com, news.radio-t.com) becomes a
// distinct "node" in the dashboard, replacing the per-rlb-redirect-target split.
//
// A Host header matching the reserved AllNode value (literally "all") is mapped to a
// safe placeholder so Candle.Update does not iterate over ["all", "all"] and double-count
// the request. The reserved-dest check on POST /api/insert catches this on the HTTP
// path; here we remap silently because nginx cannot know about rlb-stats internals.
func (p Parsed) ToRecord() store.LogRecord {
	host := p.Host
	if host == store.AllNode {
		host = "all-host"
	}
	return store.LogRecord{
		FromIP:   p.ClientIP,
		FileName: p.Path,
		DestHost: host,
		Date:     p.Date,
		Bytes:    p.Bytes,
		Country:  p.Country,
	}
}

// splitRequest extracts method and path from "$request" (e.g. "GET /foo?x=1 HTTP/2.0").
// The query string is dropped — podcast stats key by file path and query-string noise
// (CDN cache busters, player analytics) would fragment the Files map.
func splitRequest(req string) (method, path string, ok bool) {
	parts := strings.SplitN(req, " ", 3)
	if len(parts) < 2 {
		return "", "", false
	}
	method = parts[0]
	path = parts[1]
	if idx := strings.IndexByte(path, '?'); idx >= 0 {
		path = path[:idx]
	}
	return method, path, true
}

// resolveClientIP prefers the real client IP from CF-Connecting-IP; falls back to
// $remote_addr when the header is missing or dashed. Keeps the parser usable both
// behind Cloudflare and on a bare-metal nginx.
func resolveClientIP(cfIP, remote string) string {
	if cfIP != "" && cfIP != "-" {
		return cfIP
	}
	return remote
}

// resolveCountry normalises Cloudflare's CF-IPCountry: "-", "XX" (unknown) and "T1"
// (Tor exit) collapse to empty so they do not pollute the top-countries panel.
func resolveCountry(c string) string {
	switch c {
	case "", "-", "XX", "T1":
		return ""
	}
	return c
}

// truncate shortens an error message when a very long log line fails to parse so the
// WARN/DEBUG log does not dump a megabyte of input.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
