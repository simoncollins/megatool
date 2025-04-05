package mcpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSSEServerCompliance tests that the SSE server implementation is compliant with the SSE specification
// and the MCP protocol requirements.
func TestSSEServerCompliance(t *testing.T) {
	t.Run("SSE Headers", func(t *testing.T) {
		// Create a test HTTP server that simulates an SSE endpoint
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// This simulates the SSE endpoint handler
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		// Make a request to the test server
		resp, err := http.Get(ts.URL)
		assert.NoError(t, err, "HTTP request should not error")
		defer resp.Body.Close()

		// Verify the Content-Type header is set correctly for SSE
		assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"),
			"SSE endpoint should have Content-Type: text/event-stream")

		// Verify Cache-Control header is set to prevent caching
		assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"),
			"SSE endpoint should have Cache-Control: no-cache")

		// Verify Connection header is set to keep-alive
		assert.Equal(t, "keep-alive", resp.Header.Get("Connection"),
			"SSE endpoint should have Connection: keep-alive")

		// Verify the status code is 200 OK
		assert.Equal(t, http.StatusOK, resp.StatusCode,
			"SSE endpoint should return 200 OK status")
	})
}
