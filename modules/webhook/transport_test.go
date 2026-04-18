package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPTransport_Send(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if r.Header.Get("content-type") != "application/json" {
			t.Fatalf("unexpected content-type: %s", r.Header.Get("content-type"))
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tr := &HTTPTransport{
		Endpoint: server.URL,
		Timeout:  time.Second,
	}
	err := tr.Send(context.Background(), []byte(`{"k":"v"}`))
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if got["k"] != "v" {
		t.Fatalf("unexpected payload: %#v", got)
	}
}
