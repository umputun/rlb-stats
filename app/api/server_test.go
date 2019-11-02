package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/rlb-stats/app/store"
)

var storedCandle = store.Candle{
	Nodes: map[string]store.Info{
		"n6.radio-t.com": {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		"all":            {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
	},
	StartMinute: time.Time{},
}

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
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	assert.Equal(t, `{"details":"error details 123456","error":"error 500"}`+"\n", string(body))
}

func TestServer(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	startTime := time.Time{}.Format(time.RFC3339)
	endTime := time.Unix(0, 0).Format(time.RFC3339)
	var testData = []struct {
		url          string
		responseCode int
		candles      []store.Candle
		result       string
	}{
		{url: "/api/candle", responseCode: http.StatusBadRequest,
			result: "{\"details\":\"\",\"error\":\"no 'from' field passed\"}\n"},
		{url: "/api/candle?from=bad", responseCode: http.StatusExpectationFailed,
			result: "{\"details\":\"can't parse 'from' field\",\"error\":\"parsing time \\\"bad\\\" as \\\"2006-01-02T15:04:05Z07:00\\\": cannot parse \\\"bad\\\" as \\\"2006\\\"\"}\n"},
		{url: fmt.Sprintf("/api/candle?from=%v&to=bad", startTime), responseCode: http.StatusExpectationFailed,
			result: "{\"details\":\"can't parse 'to' field\",\"error\":\"parsing time \\\"bad\\\" as \\\"2006-01-02T15:04:05Z07:00\\\": cannot parse \\\"bad\\\" as \\\"2006\\\"\"}\n"},
		{url: fmt.Sprintf("/api/candle?from=%v&aggregate=bad", startTime), responseCode: http.StatusExpectationFailed,
			result: "{\"details\":\"can't parse 'aggregate' field\",\"error\":\"time: invalid duration bad\"}\n"},
		{url: fmt.Sprintf("/api/candle?from=%v&to=%v", url.QueryEscape(endTime), url.QueryEscape(endTime)), responseCode: http.StatusOK,
			result: "[]\n"},
		{url: fmt.Sprintf("/api/candle?from=%v", startTime), responseCode: http.StatusOK,
			candles: []store.Candle{storedCandle}},
		{url: fmt.Sprintf("/api/candle?from=%v&to=%v&aggregate=10m", startTime, url.QueryEscape(endTime)), responseCode: http.StatusOK,
			candles: []store.Candle{storedCandle}},
	}
	client := http.Client{}
	for i, x := range testData {
		req, err := http.NewRequest(http.MethodGet, ts.URL+x.url, nil)
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

func startupT(t *testing.T) (ts *httptest.Server, srv *Server, teardown func()) {
	file, err := ioutil.TempFile("/tmp/", "bolt_test.bd.")
	assert.Nil(t, err, "created temp file")
	storage, err := store.NewBolt(file.Name())
	assert.Nil(t, err, "engine created")
	assert.Nil(t, storage.Save(storedCandle), "saved fine")

	srv = &Server{
		address: "127.0.0.1",
		Engine:  storage,
		Port:    9999,
		Version: "test_version",
	}

	ts = httptest.NewServer(srv.routes())

	teardown = func() {
		ts.Close()
		_ = os.Remove(file.Name())
	}

	return ts, srv, teardown
}
