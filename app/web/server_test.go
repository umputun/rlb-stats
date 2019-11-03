package web

import (
	"bytes"
	"encoding/json"
	"errors"
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

func TestSendErrorJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/error" {
			t.Log("http err request", r.URL)
			sendErrorJSON(w, r, 500, errors.New("error 500"), "error details 123456")
			return
		}
		w.WriteHeader(404)
	}))

	defer ts.Close()

	resp, err := http.Get(ts.URL + "/error")
	require.Nil(t, err)

	body, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)
	assert.Equal(t, 500, resp.StatusCode)
	assert.NoError(t, resp.Body.Close())

	assert.Equal(t, `{"details":"error details 123456","error":"error 500"}`+"\n", string(body))
}

func TestServerUI(t *testing.T) {
	goodServer, _, goodTeardown := startupT(t, false)
	defer goodTeardown()
	badServer, _, badTeardown := startupT(t, true)
	defer badTeardown()

	var testData = []struct {
		ts           *httptest.Server
		url          string
		responseCode int
	}{
		{ts: goodServer, url: "/", responseCode: http.StatusOK},
		{ts: goodServer, url: "/file_stats", responseCode: http.StatusUnprocessableEntity},
		{ts: goodServer, url: "/file_stats?filename=test", responseCode: http.StatusOK},
		{ts: goodServer, url: "/chart", responseCode: http.StatusBadRequest},
		{ts: badServer, url: "/chart", responseCode: http.StatusInternalServerError},
		{ts: badServer, url: "/", responseCode: http.StatusInternalServerError},
		{ts: badServer, url: "/file_stats?filename=test", responseCode: http.StatusInternalServerError},
	}
	client := http.Client{}
	for i, x := range testData {
		req, err := http.NewRequest(http.MethodGet, x.ts.URL+x.url, nil)
		require.NoError(t, err, i)
		b, err := client.Do(req)
		require.NoError(t, err, i)
		body, err := ioutil.ReadAll(b.Body)
		require.NoError(t, err, i)
		assert.Equal(t, x.responseCode, b.StatusCode, "case %d: %v", i, string(body))
	}
}

func TestServerAPI(t *testing.T) {
	goodServer, _, goodTeardown := startupT(t, false)
	defer goodTeardown()
	badServer, _, badTeardown := startupT(t, true)
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
			result: "{\"details\":\"\",\"error\":\"no 'from' field passed\"}\n"},
		{ts: goodServer, url: "/api/candle?from=bad", responseCode: http.StatusExpectationFailed,
			result: "{\"details\":\"can't parse 'from' field\",\"error\":\"parsing time \\\"bad\\\" as \\\"2006-01-02T15:04:05Z07:00\\\": cannot parse \\\"bad\\\" as \\\"2006\\\"\"}\n"},
		{ts: goodServer, url: fmt.Sprintf("/api/candle?from=%v&to=bad", startTime), responseCode: http.StatusExpectationFailed,
			result: "{\"details\":\"can't parse 'to' field\",\"error\":\"parsing time \\\"bad\\\" as \\\"2006-01-02T15:04:05Z07:00\\\": cannot parse \\\"bad\\\" as \\\"2006\\\"\"}\n"},
		{ts: goodServer, url: fmt.Sprintf("/api/candle?from=%v&aggregate=bad", startTime), responseCode: http.StatusExpectationFailed,
			result: "{\"details\":\"can't parse 'aggregate' field\",\"error\":\"time: invalid duration bad\"}\n"},
		{ts: goodServer, url: fmt.Sprintf("/api/candle?from=%v&to=%v", startTime, startTime), responseCode: http.StatusOK,
			result: "[]\n"},
		{ts: goodServer, url: fmt.Sprintf("/api/candle?from=%v", startTime), responseCode: http.StatusOK,
			candles: []store.Candle{storedCandle}},
		{ts: badServer, url: fmt.Sprintf("/api/candle?from=%v&to=%v&aggregate=5m", startTime, url.QueryEscape(endTime)), responseCode: http.StatusBadRequest,
			result: "{\"details\":\"can't load candles\",\"error\":\"test error\"}\n"},
		{ts: goodServer, url: "/api/insert", responseCode: http.StatusBadRequest, method: http.MethodPost,
			result: "{\"details\":\"Problem decoding JSON\",\"error\":\"EOF\"}\n"},
		{ts: goodServer, url: "/api/insert", responseCode: http.StatusBadRequest, method: http.MethodPost,
			body:   bytes.NewReader([]byte(`{}`)),
			result: "{\"details\":\"ts\",\"error\":\"missing field in JSON\"}\n"},
		{ts: goodServer, url: "/api/insert", responseCode: http.StatusBadRequest, method: http.MethodPost,
			body:   bytes.NewReader([]byte(`{"ts":"1970-01-01T01:01:00+01:00"}`)),
			result: "{\"details\":\"dest\",\"error\":\"missing field in JSON\"}\n"},
		{ts: goodServer, url: "/api/insert", responseCode: http.StatusBadRequest, method: http.MethodPost,
			body:   bytes.NewReader([]byte(`{"ts":"1970-01-01T01:01:00+01:00","dest":"test"}}`)),
			result: "{\"details\":\"file_name\",\"error\":\"missing field in JSON\"}\n"},
		{ts: goodServer, url: "/api/insert", responseCode: http.StatusBadRequest, method: http.MethodPost,
			body:   bytes.NewReader([]byte(`{"ts":"1970-01-01T01:01:00+01:00","file_name":"rt_test.mp3","dest":"test"}`)),
			result: "{\"details\":\"from_ip\",\"error\":\"missing field in JSON\"}\n"},
		{ts: badServer, url: "/api/insert", responseCode: http.StatusOK, method: http.MethodPost,
			body:   bytes.NewReader([]byte(`{"from_ip":"127.0.0.1","file_name":"rt_test.mp3","dest":"new_node","ts":"1970-01-01T01:01:00+01:00"}`)),
			result: "{\"result\":\"ok\"}\n"},
		{ts: badServer, url: "/api/insert", responseCode: http.StatusInternalServerError, method: http.MethodPost,
			body:   bytes.NewReader([]byte(`{"from_ip":"127.0.0.1","file_name":"rt_test.mp3","dest":"new_node","ts":"1970-01-01T01:00:00+01:00"}`)),
			result: "{\"details\":\"Problem saving LogRecord\",\"error\":\"test error\"}\n"},
	}
	client := http.Client{}
	for i, x := range testData {
		if x.method == "" {
			x.method = http.MethodGet
		}
		req, err := http.NewRequest(x.method, x.ts.URL+x.url, x.body)
		require.NoError(t, err, i)
		b, err := client.Do(req)
		require.NoError(t, err, i)
		body, err := ioutil.ReadAll(b.Body)
		require.NoError(t, err, i)
		if x.result != "" {
			assert.Equal(t, x.result, string(body), i)
		}
		if x.candles != nil {
			var candles []store.Candle
			err = json.Unmarshal(body, &candles)
			if err != nil {
				require.Nil(t, string(body), "problem parsing response body, case %d", i)
			}
			assert.Equal(t, x.candles, candles, i)
		}
		assert.Equal(t, x.responseCode, b.StatusCode, "case %d: %v", i, string(body))
	}
}

func startupT(t *testing.T, badEngine bool) (ts *httptest.Server, srv *Server, teardown func()) {
	storage, engineTeardown := startupEngine(t, badEngine)

	srv = &Server{
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

	return ts, srv, teardown
}
