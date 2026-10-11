package nginx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mkParsed(method string, status int, path, host string) Parsed {
	return Parsed{Method: method, Status: status, Path: path, Host: host}
}

func TestFilterKeep(t *testing.T) {
	tests := []struct {
		name    string
		methods []string
		status  []int
		paths   []string
		hosts   []string
		in      Parsed
		keep    bool
	}{
		{
			name:   "defaults: GET-only, GET /x.mp3 kept",
			in:     mkParsed("GET", 200, "/x.mp3", "cdn"),
			keep:   true,
		},
		{
			name:   "defaults: HEAD /x.mp3 dropped",
			in:     mkParsed("HEAD", 200, "/x.mp3", "cdn"),
			keep:   false, // the regression trap: HEAD probes must not count
		},
		{
			name:    "methods allowlist: HEAD kept when explicit",
			methods: []string{"GET", "HEAD"},
			in:      mkParsed("HEAD", 200, "/x.mp3", "cdn"),
			keep:    true,
		},
		{
			name:   "status allowlist: 200,206 keeps 206",
			status: []int{200, 206},
			in:     mkParsed("GET", 206, "/x.mp3", "cdn"),
			keep:   true,
		},
		{
			name:   "status allowlist: 200,206 drops 304",
			status: []int{200, 206},
			in:     mkParsed("GET", 304, "/x.html", "radio-t.com"),
			keep:   false,
		},
		{
			name:  "path glob: *.mp3 keeps /rt_podcast.mp3",
			paths: []string{"*.mp3"},
			in:    mkParsed("GET", 200, "/rt_podcast.mp3", "cdn"),
			keep:  true,
		},
		{
			name:  "path glob: *.mp3 drops /site-api/last/1",
			paths: []string{"*.mp3"},
			in:    mkParsed("GET", 200, "/site-api/last/1", "radio-t.com"),
			keep:  false,
		},
		{
			name:  "path glob: multiple patterns",
			paths: []string{"*.mp3", "*.ogg"},
			in:    mkParsed("GET", 200, "/thing.ogg", "cdn"),
			keep:  true,
		},
		{
			name:  "path glob: match on basename not full path",
			paths: []string{"*.mp3"},
			in:    mkParsed("GET", 200, "/path/to/file.mp3", "cdn"),
			keep:  true,
		},
		{
			name:  "host allowlist: cdn.radio-t.com keeps cdn.radio-t.com",
			hosts: []string{"cdn.radio-t.com"},
			in:    mkParsed("GET", 200, "/x.mp3", "cdn.radio-t.com"),
			keep:  true,
		},
		{
			name:  "host allowlist: cdn.radio-t.com drops news.radio-t.com",
			hosts: []string{"cdn.radio-t.com"},
			in:    mkParsed("GET", 200, "/x.mp3", "news.radio-t.com"),
			keep:  false,
		},
		{
			name:    "default filter against umputun's seven lines: HEAD and 304 dropped",
			methods: nil,                 // GET-only
			status:  []int{200, 206},     // podcast-focused
			paths:   []string{"*.mp3"},   // podcast-focused
			in:      mkParsed("GET", 304, "/p/2026/04/18/podcast-1009/", "radio-t.com"),
			keep:    false,
		},
		{
			name:    "default filter: mp3 206 kept",
			methods: nil,
			status:  []int{200, 206},
			paths:   []string{"*.mp3"},
			in:      mkParsed("GET", 206, "/rt_podcast1010.mp3", "cdn.radio-t.com"),
			keep:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f, err := NewFilter(tc.methods, tc.status, tc.paths, tc.hosts)
			require.NoError(t, err)
			assert.Equal(t, tc.keep, f.Keep(tc.in))
		})
	}
}

func TestNewFilterRejectsBadGlob(t *testing.T) {
	// path.Match returns ErrBadPattern for an unclosed character class; validating
	// eagerly keeps a typo in a flag from flooding DEBUG for every single log line.
	_, err := NewFilter(nil, nil, []string{"[abc"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path glob")
}
