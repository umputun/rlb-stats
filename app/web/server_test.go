package web

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerWithoutApi(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	var testData = []struct {
		url          string
		responseCode int
	}{
		{url: "/", responseCode: http.StatusInternalServerError},
		{url: "/file_stats", responseCode: http.StatusUnprocessableEntity},
		{url: "/file_stats?filename=test", responseCode: http.StatusInternalServerError},
		{url: "/chart", responseCode: http.StatusInternalServerError},
	}
	client := http.Client{}
	for i, x := range testData {
		req, err := http.NewRequest(http.MethodGet, ts.URL+x.url, nil)
		require.NoError(t, err, i)
		b, err := client.Do(req)
		require.NoError(t, err, i)
		body, err := ioutil.ReadAll(b.Body)
		require.NoError(t, err, i)
		assert.Equal(t, x.responseCode, b.StatusCode, "case %d: %v", i, string(body))
	}
}

func startupT(t *testing.T) (ts *httptest.Server, srv *Server, teardown func()) {
	srv = &Server{
		address: "127.0.0.1",
		Port:    9999,
		APIPort: 9998,
	}

	ts = httptest.NewServer(srv.routes())

	teardown = func() {
		ts.Close()
	}

	return ts, srv, teardown
}
