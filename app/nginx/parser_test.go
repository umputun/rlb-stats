package nginx

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// umputun's seven real log lines from Telegram 2026-04-26 msg 496162, with the three
// new CF fields appended as if nginx had already been reconfigured to the new log_format.
// Country codes are chosen so TestParseAllRealLines can assert the full map.
var realLines = []struct {
	line        string
	wantMethod  string
	wantStatus  int
	wantPath    string
	wantHost    string
	wantIP      string
	wantCountry string
	wantBytes   int64
}{
	{
		line: `108.162.220.92 - - [26/Apr/2026:21:52:30 +0000] "GET /p/2026/04/18/podcast-1009/ HTTP/2.0" 304 0 "radio-t.com" "https://radio-t.com/" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.4 Safari/605.1.15" https "92.47.111.1" "US" "-"`,
		wantMethod: "GET", wantStatus: 304, wantPath: "/p/2026/04/18/podcast-1009/",
		wantHost: "radio-t.com", wantIP: "92.47.111.1", wantCountry: "US", wantBytes: 0,
	},
	{
		line: `108.162.220.92 - - [26/Apr/2026:21:52:30 +0000] "GET /p/2026/04/21/prep-1010/ HTTP/2.0" 304 0 "radio-t.com" "https://radio-t.com/" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.4 Safari/605.1.15" https "92.47.111.1" "US" "-"`,
		wantMethod: "GET", wantStatus: 304, wantPath: "/p/2026/04/21/prep-1010/",
		wantHost: "radio-t.com", wantIP: "92.47.111.1", wantCountry: "US", wantBytes: 0,
	},
	{
		line: `108.162.220.92 - - [26/Apr/2026:21:52:30 +0000] "GET /p/2026/04/25/podcast-1010/ HTTP/2.0" 304 0 "radio-t.com" "https://radio-t.com/" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.4 Safari/605.1.15" https "92.47.111.1" "US" "-"`,
		wantMethod: "GET", wantStatus: 304, wantPath: "/p/2026/04/25/podcast-1010/",
		wantHost: "radio-t.com", wantIP: "92.47.111.1", wantCountry: "US", wantBytes: 0,
	},
	{
		line: `162.158.95.94 - - [26/Apr/2026:21:52:31 +0000] "GET /site-api/last/1 HTTP/2.0" 200 1095 "radio-t.com" "-" "Go-http-client/2.0" https "-" "-" "-"`,
		wantMethod: "GET", wantStatus: 200, wantPath: "/site-api/last/1",
		wantHost: "radio-t.com", wantIP: "162.158.95.94", wantCountry: "", wantBytes: 1095,
	},
	{
		// regression trap: same remote_addr (CF edge) as the second mp3 line below, but
		// different cf_connecting_ip. Both must land as distinct downloads because dedup
		// keys on client IP, not edge IP. Parsing must therefore use cfip.
		line: `108.162.220.92 - - [26/Apr/2026:21:52:32 +0000] "GET /rt_podcast1010.mp3 HTTP/2.0" 206 2097152 "cdn.radio-t.com" "-" "Podcasts/4025.510.6 CFNetwork/3860.500.112 Darwin/25.4.0" https "31.46.245.136" "DE" "bytes=0-2097151"`,
		wantMethod: "GET", wantStatus: 206, wantPath: "/rt_podcast1010.mp3",
		wantHost: "cdn.radio-t.com", wantIP: "31.46.245.136", wantCountry: "DE", wantBytes: 2097152,
	},
	{
		line: `108.162.220.92 - - [26/Apr/2026:21:52:32 +0000] "GET /rt_podcast1010.mp3 HTTP/2.0" 200 139264 "cdn.radio-t.com" "-" "atc/1.0 watchOS/26.3" https "109.196.194.69" "PL" "-"`,
		wantMethod: "GET", wantStatus: 200, wantPath: "/rt_podcast1010.mp3",
		wantHost: "cdn.radio-t.com", wantIP: "109.196.194.69", wantCountry: "PL", wantBytes: 139264,
	},
	{
		line: `20.215.220.195 - - [26/Apr/2026:21:52:33 +0000] "GET /post/schedule-tasks-on-the-web-claude-code-docs HTTP/2.0" 200 1197 "news.radio-t.com" "-" "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko); compatible; ChatGPT-User/1.0; +https://openai.com/bot" https "-" "US" "-"`,
		wantMethod: "GET", wantStatus: 200, wantPath: "/post/schedule-tasks-on-the-web-claude-code-docs",
		wantHost: "news.radio-t.com", wantIP: "20.215.220.195", wantCountry: "US", wantBytes: 1197,
	},
}

func TestParseAllRealLines(t *testing.T) {
	for i, tc := range realLines {
		t.Run("line_"+tc.wantPath, func(t *testing.T) {
			got, err := Parse(tc.line)
			require.NoError(t, err, "line %d: %q", i, tc.line)
			assert.Equal(t, tc.wantMethod, got.Method)
			assert.Equal(t, tc.wantStatus, got.Status)
			assert.Equal(t, tc.wantPath, got.Path)
			assert.Equal(t, tc.wantHost, got.Host)
			assert.Equal(t, tc.wantIP, got.ClientIP)
			assert.Equal(t, tc.wantCountry, got.Country)
			assert.Equal(t, tc.wantBytes, got.Bytes)
			assert.Equal(t, 2026, got.Date.Year())
			assert.Equal(t, time.April, got.Date.Month())
		})
	}
}

func TestParseRejectsOldFormat(t *testing.T) {
	// umputun's unmodified pre-update line — eleven fields, no CF fields. If this parsed
	// successfully the parser would silently feed CF-less records into aggregation,
	// defeating the whole reason for the log_format update.
	oldLine := `108.162.220.92 - - [26/Apr/2026:21:52:32 +0000] "GET /rt_podcast1010.mp3 HTTP/2.0" 206 2097152 "cdn.radio-t.com" "-" "Podcasts/4025.510.6 CFNetwork/3860.500.112 Darwin/25.4.0" https`
	_, err := Parse(oldLine)
	require.Error(t, err, "old-format line must not parse")
	assert.ErrorIs(t, err, ErrBadLine)
}

func TestParseEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantErr bool
		check   func(t *testing.T, p Parsed)
	}{
		{
			name:    "empty line",
			line:    "",
			wantErr: true,
		},
		{
			name:    "random garbage",
			line:    "hello world",
			wantErr: true,
		},
		{
			name: "query string stripped from path",
			line: `10.0.0.1 - - [26/Apr/2026:21:52:30 +0000] "GET /a.mp3?token=abc&cb=1 HTTP/2.0" 200 100 "cdn" "-" "x" https "-" "-" "-"`,
			check: func(t *testing.T, p Parsed) {
				assert.Equal(t, "/a.mp3", p.Path)
			},
		},
		{
			name: "cf_connecting_ip empty falls back to remote_addr",
			line: `10.0.0.1 - - [26/Apr/2026:21:52:30 +0000] "GET /a.mp3 HTTP/2.0" 200 100 "cdn" "-" "x" https "" "-" "-"`,
			check: func(t *testing.T, p Parsed) {
				assert.Equal(t, "10.0.0.1", p.ClientIP, "empty CF IP must fall back to remote")
			},
		},
		{
			name: "cf_connecting_ip dash falls back to remote_addr",
			line: `10.0.0.1 - - [26/Apr/2026:21:52:30 +0000] "GET /a.mp3 HTTP/2.0" 200 100 "cdn" "-" "x" https "-" "-" "-"`,
			check: func(t *testing.T, p Parsed) {
				assert.Equal(t, "10.0.0.1", p.ClientIP)
			},
		},
		{
			name: "cf_ipcountry XX collapses to empty",
			line: `10.0.0.1 - - [26/Apr/2026:21:52:30 +0000] "GET /a.mp3 HTTP/2.0" 200 100 "cdn" "-" "x" https "-" "XX" "-"`,
			check: func(t *testing.T, p Parsed) {
				assert.Equal(t, "", p.Country, "XX must not pollute the top-countries panel")
			},
		},
		{
			name: "HEAD method parsed",
			line: `10.0.0.1 - - [26/Apr/2026:21:52:30 +0000] "HEAD /a.mp3 HTTP/2.0" 200 0 "cdn" "-" "x" https "-" "-" "-"`,
			check: func(t *testing.T, p Parsed) {
				assert.Equal(t, "HEAD", p.Method)
				assert.Equal(t, int64(0), p.Bytes)
			},
		},
		{
			name: "range header preserved on parsed",
			line: `10.0.0.1 - - [26/Apr/2026:21:52:30 +0000] "GET /a.mp3 HTTP/2.0" 206 1024 "cdn" "-" "x" https "-" "-" "bytes=0-1023"`,
			check: func(t *testing.T, _ Parsed) {
				// http_range is logged but discarded for now; just assert the line parses
				// so the format stays stable when probe-vs-play analysis is added later.
			},
		},
		{
			name:    "non-integer status",
			line:    `10.0.0.1 - - [26/Apr/2026:21:52:30 +0000] "GET /a.mp3 HTTP/2.0" abc 100 "cdn" "-" "x" https "-" "-" "-"`,
			wantErr: true,
		},
		{
			name:    "malformed time",
			line:    `10.0.0.1 - - [notatime] "GET /a.mp3 HTTP/2.0" 200 100 "cdn" "-" "x" https "-" "-" "-"`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := Parse(tc.line)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.check != nil {
				tc.check(t, p)
			}
		})
	}
}

func TestParseErrorsBundleOriginalLine(t *testing.T) {
	// a DEBUG log should name why the line was dropped; the error must carry both the
	// sentinel (so callers can errors.Is for metrics) and a readable extract of the input
	line := "not an nginx line at all"
	_, err := Parse(line)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBadLine))
	assert.Contains(t, err.Error(), "not an nginx")
}

func TestParsedToRecordRemapsReservedHost(t *testing.T) {
	// If a client sends Host: all and nginx logs it verbatim, DestHost would collide
	// with the synthetic AllNode and double-count the request (Volume=2, Bytes=2x,
	// Countries{code}=2) because Candle.Update iterates [DestHost, AllNode] and both
	// would be "all". The parser remaps this reserved value before building the record.
	p := Parsed{Method: "GET", Status: 200, Path: "/a.mp3", Host: "all", ClientIP: "1.1.1.1", Bytes: 100, Country: "DE"}
	r := p.ToRecord()
	assert.NotEqual(t, "all", r.DestHost, "reserved host must be remapped off AllNode")
	assert.Equal(t, "all-host", r.DestHost)
}

func TestParsedToRecord(t *testing.T) {
	p := Parsed{
		Method:   "GET",
		Status:   200,
		Path:     "/a.mp3",
		Host:     "cdn.radio-t.com",
		ClientIP: "1.2.3.4",
		Country:  "US",
		Bytes:    1000,
		Date:     time.Date(2026, time.April, 26, 21, 52, 30, 0, time.UTC),
	}
	r := p.ToRecord()
	assert.Equal(t, "1.2.3.4", r.FromIP)
	assert.Equal(t, "/a.mp3", r.FileName)
	assert.Equal(t, "cdn.radio-t.com", r.DestHost, "Host becomes the node")
	assert.Equal(t, int64(1000), r.Bytes)
	assert.Equal(t, "US", r.Country)
	assert.Equal(t, p.Date, r.Date)
}
