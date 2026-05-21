package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func mkResp(status int, hdr map[string]string) *http.Response {
	h := http.Header{}
	for k, v := range hdr {
		h.Set(k, v)
	}
	return &http.Response{StatusCode: status, Header: h}
}

func TestIsRetryable(t *testing.T) {
	cases := []struct {
		name   string
		status int
		hdr    map[string]string
		method string
		want   bool
	}{
		{"429 retried", 429, nil, "GET", true},
		{"403 with retry-after retried", 403, map[string]string{"Retry-After": "1"}, "POST", true},
		{"bare 403 not retried", 403, nil, "GET", false},
		{"500 GET retried", 500, nil, "GET", true},
		{"500 POST not retried (dup-create risk)", 500, nil, "POST", false},
		{"502 PATCH retried", 502, nil, "PATCH", true},
		{"200 not retried", 200, nil, "GET", false},
		{"404 not retried", 404, nil, "GET", false},
		{"422 not retried", 422, nil, "POST", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isRetryable(mkResp(tc.status, tc.hdr), tc.method); got != tc.want {
				t.Errorf("isRetryable(%d,%s)=%v want %v", tc.status, tc.method, got, tc.want)
			}
		})
	}
}

func TestRetryDelay(t *testing.T) {
	if d := retryDelay(mkResp(429, map[string]string{"Retry-After": "5"}), 0); d != 5*time.Second {
		t.Errorf("Retry-After: got %v want 5s", d)
	}
	if d := retryDelay(mkResp(429, map[string]string{"Retry-After": "9999"}), 0); d != maxBackoff {
		t.Errorf("Retry-After cap: got %v want %v", d, maxBackoff)
	}
	reset := time.Now().Add(10 * time.Second).Unix()
	d := retryDelay(mkResp(403, map[string]string{
		"X-RateLimit-Remaining": "0",
		"X-RateLimit-Reset":     strconv.FormatInt(reset, 10),
	}), 0)
	if d < 8*time.Second || d > 11*time.Second {
		t.Errorf("X-RateLimit-Reset: got %v want ~10s", d)
	}
	for attempt, want := range map[int]time.Duration{0: time.Second, 1: 2 * time.Second, 2: 4 * time.Second} {
		if d := retryDelay(mkResp(500, nil), attempt); d != want {
			t.Errorf("backoff attempt %d: got %v want %v", attempt, d, want)
		}
	}
}

func TestCapDelay(t *testing.T) {
	if capDelay(0) != time.Second {
		t.Errorf("floor: got %v want 1s", capDelay(0))
	}
	if capDelay(2*time.Hour) != maxBackoff {
		t.Errorf("ceil: got %v want %v", capDelay(2*time.Hour), maxBackoff)
	}
	if capDelay(5*time.Second) != 5*time.Second {
		t.Errorf("passthrough: got %v want 5s", capDelay(5*time.Second))
	}
}

// withBaseURL points the client base URL at an httptest server for the test.
func withBaseURL(t *testing.T, url string) {
	old := baseURL
	baseURL = url
	t.Cleanup(func() { baseURL = old })
}

func TestDoRequestRetriesThenSucceeds(t *testing.T) {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&n, 1) == 1 {
			w.Header().Set("Retry-After", "0") // floored to 1s by capDelay
			w.WriteHeader(429)
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	withBaseURL(t, srv.URL)

	data, err := New("t", "o", "r", false).Get(context.Background(), "issues", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(&n); got != 2 {
		t.Errorf("attempts=%d want 2 (one retry)", got)
	}
	if string(data) != `{"ok":true}` {
		t.Errorf("body=%s", data)
	}
}

func TestDoRequestBare403NoRetry(t *testing.T) {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&n, 1)
		w.WriteHeader(403)
		w.Write([]byte(`{"message":"Forbidden"}`))
	}))
	defer srv.Close()
	withBaseURL(t, srv.URL)

	if _, err := New("t", "o", "r", false).Get(context.Background(), "issues", nil); err == nil {
		t.Fatal("expected error")
	}
	if got := atomic.LoadInt32(&n); got != 1 {
		t.Errorf("attempts=%d want 1 (bare 403 not retried)", got)
	}
}

func TestDoRequestPost5xxNoRetry(t *testing.T) {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&n, 1)
		w.WriteHeader(500)
	}))
	defer srv.Close()
	withBaseURL(t, srv.URL)

	if _, err := New("t", "o", "r", false).Post(context.Background(), "issues", map[string]any{"x": 1}); err == nil {
		t.Fatal("expected error")
	}
	if got := atomic.LoadInt32(&n); got != 1 {
		t.Errorf("attempts=%d want 1 (POST 5xx not retried)", got)
	}
}

func TestDoRequestBodyResentOnRetry(t *testing.T) {
	var n int32
	bodies := make([]string, 0, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		if atomic.AddInt32(&n, 1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(429) // rate limit → retried for any method, incl. PATCH
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	withBaseURL(t, srv.URL)

	if _, err := New("t", "o", "r", false).Patch(context.Background(), "issues/1", map[string]any{"title": "x"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bodies) != 2 || bodies[0] != bodies[1] || bodies[0] == "" {
		t.Errorf("body not re-sent identically across retry: %#v", bodies)
	}
}
