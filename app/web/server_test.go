package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/rlb-stats/app/store"
)

func TestServerDashboard(t *testing.T) {
	goodServer, goodTeardown := startupT(t, false)
	defer goodTeardown()

	client := http.Client{}

	t.Run("GET / returns full HTML page", func(t *testing.T) {
		resp, err := client.Get(goodServer.URL + "/")
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, resp.Body.Close())
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
		assert.Contains(t, string(body), "<!DOCTYPE html>")
		assert.Contains(t, string(body), "<title>RLB Stats</title>")
		assert.Contains(t, string(body), "id=\"dashboard\"")
		assert.Contains(t, string(body), "Downloads")
	})

	t.Run("GET /fragment/dashboard returns HTML fragment", func(t *testing.T) {
		resp, err := client.Get(goodServer.URL + "/fragment/dashboard?period=1h")
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, resp.Body.Close())
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
		assert.NotContains(t, string(body), "<!DOCTYPE html>")
		assert.NotContains(t, string(body), "<html")
		assert.Contains(t, string(body), "Downloads")
	})

	t.Run("GET /fragment/dashboard?period=all works with TimeRange", func(t *testing.T) {
		resp, err := client.Get(goodServer.URL + "/fragment/dashboard?period=all")
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, resp.Body.Close())
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, string(body), "Downloads")
	})

	t.Run("GET /fragment/dashboard with invalid period returns 400", func(t *testing.T) {
		resp, err := client.Get(goodServer.URL + "/fragment/dashboard?period=99h")
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("GET / with default period renders 24h", func(t *testing.T) {
		resp, err := client.Get(goodServer.URL + "/")
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, resp.Body.Close())
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		// the 24h button should have aria-current="true"
		assert.Contains(t, string(body), "aria-current")
	})

	t.Run("GET /static/charts.js returns JS file", func(t *testing.T) {
		resp, err := client.Get(goodServer.URL + "/static/charts.js")
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, resp.Body.Close())
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, string(body), "initCharts")
		assert.Contains(t, string(body), "htmx:afterSettle")
	})
}

func TestServerDashboardBadEngine(t *testing.T) {
	badServer, badTeardown := startupT(t, true)
	defer badTeardown()

	client := http.Client{}

	t.Run("GET / with bad engine returns 400", func(t *testing.T) {
		resp, err := client.Get(badServer.URL + "/")
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("GET /fragment/dashboard with bad engine returns 400", func(t *testing.T) {
		resp, err := client.Get(badServer.URL + "/fragment/dashboard?period=all")
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
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
		{ts: goodServer, url: "/api/candle?from=bad", responseCode: http.StatusBadRequest,
			result: "{\"error\":\"can't parse 'from' field\"}\n"},
		{ts: goodServer, url: fmt.Sprintf("/api/candle?from=%v&to=bad", startTime), responseCode: http.StatusBadRequest,
			result: "{\"error\":\"can't parse 'to' field\"}\n"},
		{ts: goodServer, url: fmt.Sprintf("/api/candle?from=%v&aggregate=bad", startTime), responseCode: http.StatusBadRequest,
			result: "{\"error\":\"can't parse 'aggregate' field\"}\n"},
		{ts: goodServer, url: fmt.Sprintf("/api/candle?from=%v&max_points=256", startTime), responseCode: http.StatusOK,
			candles: []store.Candle{storedCandle}},
		{ts: goodServer, url: fmt.Sprintf("/api/candle?from=%v&max_points=200", startTime), responseCode: http.StatusOK,
			candles: []store.Candle{storedCandle}},
		{ts: goodServer, url: fmt.Sprintf("/api/candle?from=%v&to=%v", startTime, startTime), responseCode: http.StatusOK,
			result: "[]\n"},
		{ts: goodServer, url: fmt.Sprintf("/api/candle?from=%v", startTime), responseCode: http.StatusOK,
			candles: []store.Candle{storedCandle}},
		{ts: goodServer, url: fmt.Sprintf("/api/candle?from=%v&files=bad", startTime), responseCode: http.StatusBadRequest,
			result: "{\"error\":\"can't parse 'files' field\"}\n"},
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
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			if x.method == "" {
				x.method = http.MethodGet
			}
			req, err := http.NewRequest(x.method, x.ts.URL+x.url, x.body)
			require.NoError(t, err, i)
			b, err := client.Do(req)
			require.NoError(t, err, i)
			defer b.Body.Close()
			body, err := io.ReadAll(b.Body)
			require.NoError(t, err, i)
			if x.result != "" {
				assert.Equal(t, x.result, string(body), i)
			}
			if x.candles != nil {
				var candles []store.Candle
				err = json.Unmarshal(body, &candles)
				if err != nil {
					require.NoError(t, err, "problem parsing response body: %s", string(body))
				}
				assert.Equal(t, x.candles, candles, i)
			}
			assert.Equal(t, x.responseCode, b.StatusCode, string(body))
		})
	}
}

func TestServerRunShutdown(t *testing.T) {
	storage, teardown := startupEngine(t, false)
	defer teardown()

	srv := &Server{
		address:    "127.0.0.1",
		Engine:     storage,
		Aggregator: &store.Aggregator{},
		Port:       0, // will use address from test
		Version:    "test",
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		srv.Run(ctx)
		close(done)
	}()

	// give server time to start
	time.Sleep(50 * time.Millisecond)

	// cancel context to trigger shutdown
	cancel()

	select {
	case <-done:
		// server shut down cleanly
	case <-time.After(3 * time.Second):
		t.Fatal("server did not shut down within 3 seconds")
	}
}

func startupT(t *testing.T, badEngine bool) (ts *httptest.Server, teardown func()) {
	storage, engineTeardown := startupEngine(t, badEngine)

	srv := &Server{
		address:    "127.0.0.1",
		Engine:     storage,
		Aggregator: &store.Aggregator{},
		Port:       9999,
		Version:    "test_version",
	}

	ts = httptest.NewServer(srv.routes())

	teardown = func() {
		ts.Close()
		engineTeardown()
	}

	return ts, teardown
}
