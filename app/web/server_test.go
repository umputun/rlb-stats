package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/rlb-stats/app/store"
)

func TestServerUI(t *testing.T) {
	goodServer, goodTeardown := startupT(t, false)
	defer goodTeardown()

	var testData = []struct {
		ts           *httptest.Server
		url          string
		responseCode int
	}{
		{ts: goodServer, url: "/", responseCode: http.StatusOK},
		{ts: goodServer, url: "/index.js", responseCode: http.StatusOK},
		{ts: goodServer, url: "/favicon.ico", responseCode: http.StatusOK},
		{ts: goodServer, url: "/data.js", responseCode: http.StatusOK},
		{ts: goodServer, url: "/index.html", responseCode: http.StatusOK},
	}
	client := http.Client{}
	for i, x := range testData {
		i := i
		x := x
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, x.ts.URL+x.url, nil)
			require.NoError(t, err, i)
			b, err := client.Do(req)
			require.NoError(t, err, i)
			body, err := ioutil.ReadAll(b.Body)
			require.NoError(t, b.Body.Close())
			require.NoError(t, err, i)
			assert.Equal(t, x.responseCode, b.StatusCode, string(body))
		})
	}
}

func TestServerAPI(t *testing.T) {
	goodServer, goodTeardown := startupT(t, false)
	defer goodTeardown()
	badServer, badTeardown := startupT(t, true)
	defer badTeardown()

	startTime := time.Time{}.Format(time.RFC3339)
	endTime := time.Unix(0, 0).Format(time.RFC3339)
	var testData = []struct {
		ts           *httptest.Server
		url          string
		responseCode int
		candles      []store.Candle
		result       string
		method       string
		body         io.Reader
	}{
		{ts: goodServer, url: "/api/candle", responseCode: http.StatusBadRequest,
			result: "{\"error\":\"no 'from' field passed\"}\n"},
		{ts: goodServer, url: "/api/candle?from=bad", responseCode: http.StatusExpectationFailed,
			result: "{\"error\":\"can't parse 'from' field\"}\n"},
		{ts: goodServer, url: fmt.Sprintf("/api/candle?from=%v&to=bad", startTime), responseCode: http.StatusExpectationFailed,
			result: "{\"error\":\"can't parse 'to' field\"}\n"},
		{ts: goodServer, url: fmt.Sprintf("/api/candle?from=%v&aggregate=bad", startTime), responseCode: http.StatusExpectationFailed,
			result: "{\"error\":\"can't parse 'aggregate' field\"}\n"},
		{ts: goodServer, url: fmt.Sprintf("/api/candle?from=%v&max_points=256", startTime), responseCode: http.StatusExpectationFailed,
			result: "{\"error\":\"can't parse 'max_points' field\"}\n"},
		{ts: goodServer, url: fmt.Sprintf("/api/candle?from=%v&to=%v", startTime, startTime), responseCode: http.StatusOK,
			result: "[]\n"},
		{ts: goodServer, url: fmt.Sprintf("/api/candle?from=%v", startTime), responseCode: http.StatusOK,
			candles: []store.Candle{storedCandle}},
		{ts: badServer, url: fmt.Sprintf("/api/candle?from=%v&to=%v&aggregate=5m&max_points=10", startTime, url.QueryEscape(endTime)), responseCode: http.StatusBadRequest,
			result: "{\"error\":\"can't load candles\"}\n"},
		{ts: goodServer, url: "/api/insert", responseCode: http.StatusBadRequest, method: http.MethodPost,
			result: "{\"error\":\"Problem decoding JSON\"}\n"},
		{ts: goodServer, url: "/api/insert", responseCode: http.StatusBadRequest, method: http.MethodPost,
			body:   bytes.NewReader([]byte(`{}`)),
			result: "{\"error\":\"missing field in JSON: ts\"}\n"},
		{ts: goodServer, url: "/api/insert", responseCode: http.StatusBadRequest, method: http.MethodPost,
			body:   bytes.NewReader([]byte(`{"ts":"1970-01-01T01:01:00+01:00"}`)),
			result: "{\"error\":\"missing field in JSON: dest\"}\n"},
		{ts: goodServer, url: "/api/insert", responseCode: http.StatusBadRequest, method: http.MethodPost,
			body:   bytes.NewReader([]byte(`{"ts":"1970-01-01T01:01:00+01:00","dest":"test"}}`)),
			result: "{\"error\":\"missing field in JSON: file_name\"}\n"},
		{ts: goodServer, url: "/api/insert", responseCode: http.StatusBadRequest, method: http.MethodPost,
			body:   bytes.NewReader([]byte(`{"ts":"1970-01-01T01:01:00+01:00","file_name":"rt_test.mp3","dest":"test"}`)),
			result: "{\"error\":\"missing field in JSON: from_ip\"}\n"},
		{ts: badServer, url: "/api/insert", responseCode: http.StatusOK, method: http.MethodPost,
			body:   bytes.NewReader([]byte(`{"from_ip":"127.0.0.1","file_name":"rt_test.mp3","dest":"new_node","ts":"1970-01-01T01:01:00+01:00"}`)),
			result: "{\"result\":\"ok\"}\n"},
		{ts: badServer, url: "/api/insert", responseCode: http.StatusInternalServerError, method: http.MethodPost,
			body:   bytes.NewReader([]byte(`{"from_ip":"127.0.0.1","file_name":"rt_test.mp3","dest":"new_node","ts":"1970-01-01T01:00:00+01:00"}`)),
			result: "{\"error\":\"Problem saving LogRecord\"}\n"},
	}
	client := http.Client{}
	for i, x := range testData {
		i := i
		x := x
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			if x.method == "" {
				x.method = http.MethodGet
			}
			req, err := http.NewRequest(x.method, x.ts.URL+x.url, x.body)
			require.NoError(t, err, i)
			b, err := client.Do(req)
			require.NoError(t, err, i)
			defer b.Body.Close()
			body, err := ioutil.ReadAll(b.Body)
			require.NoError(t, err, i)
			if x.result != "" {
				assert.Equal(t, x.result, string(body), i)
			}
			if x.candles != nil {
				var candles []store.Candle
				err = json.Unmarshal(body, &candles)
				if err != nil {
					require.Nil(t, string(body), "problem parsing response body")
				}
				assert.Equal(t, x.candles, candles, i)
			}
			assert.Equal(t, x.responseCode, b.StatusCode, string(body))
		})
	}
}

func startupT(t *testing.T, badEngine bool) (ts *httptest.Server, teardown func()) {
	storage, engineTeardown := startupEngine(t, badEngine)

	srv := &Server{
		address:      "127.0.0.1",
		Engine:       storage,
		Aggregator:   &store.Aggregator{},
		Port:         9999,
		Version:      "test_version",
		webappPrefix: "../../",
	}

	ts = httptest.NewServer(srv.routes())

	teardown = func() {
		ts.Close()
		engineTeardown()
	}

	return ts, teardown
}
