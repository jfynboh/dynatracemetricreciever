package dynatracemetricreceiver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandleValidMetrics(t *testing.T) {
	// consumer := consumertest.NewNop()
	// cfg := &Config{Endpoint: ":0"}
	// // r := newReceiver(cfg, consumer)

	//   req := httptest.NewRequest("POST", "/", bytes.NewBufferString("foo:1.0
	// bar:2.5
	// invalid
	// "))
	w := httptest.NewRecorder()
	// r.handle(w, req)

	res := w.Result()
	body, _ := io.ReadAll(res.Body)
	require.Equal(t, http.StatusAccepted, res.StatusCode, string(body))
}
