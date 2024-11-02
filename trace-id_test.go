package traefik_add_trace_id_header_2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateTraceId(t *testing.T) {
	testMe := &TraceIDHeader{}

	testMe.uuidGen = "4"
	got := testMe.GenerateTraceId()
	if len(got) != 36 {
		t.Fatal("Failed to return a valid UUIDv4 trace ID.")
	}

	testMe.uuidGen = "7"
	got = testMe.GenerateTraceId()
	if len(got) != 36 {
		t.Fatal("Failed to return a valid UUIDv7 trace ID.")
	}

	testMe.uuidGen = "L"
	got = testMe.GenerateTraceId()
	if len(got) != 26 {
		t.Fatal("Failed to return a valid ULID trace ID.")
	}

	testMe.uuidGen = "Z" // not valid
	got = testMe.GenerateTraceId()
	if len(got) != 0 {
		t.Fatal("Failed to return an empty trace ID for an invalid uuidGen value.")
	}
}

func TestServeHTTP(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		assertFunc func(t *testing.T) http.Handler
	}{
		{
			name:   "remote IP is trusted",
			config: &Config{},
			assertFunc: func(t *testing.T) http.Handler {
				t.Helper()
				return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					hdr := getTraceIdHeader(t, req, "X-Trace-Id")
					mustHaveLength(t, hdr, 36)
				})
			},
		},
		{
			name:   "no trace id",
			config: &Config{},
			assertFunc: func(t *testing.T) http.Handler {
				t.Helper()
				return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					hdr := getTraceIdHeader(t, req, "X-Trace-Id")
					mustHaveLength(t, hdr, 36)
				})
			},
		},
		{
			name: "custom name",
			config: &Config{
				HeaderName: "Other-Name",
			},
			assertFunc: func(t *testing.T) http.Handler {
				t.Helper()
				return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					hdr := getTraceIdHeader(t, req, "Other-Name")
					mustHaveLength(t, hdr, 36)
				})
			},
		},
		{
			name: "with prefix",
			config: &Config{
				ValuePrefix: "myorg",
			},
			assertFunc: func(t *testing.T) http.Handler {
				t.Helper()
				return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					hdr := getTraceIdHeader(t, req, "X-Trace-Id")
					mustHavePrefix(t, hdr, "myorg")
					mustHaveLength(t, hdr, 41)
				})
			},
		},
		{
			name: "custom traceid and prefix",
			config: &Config{
				ValuePrefix: "myorg",
				HeaderName:  "Other-Name",
			},
			assertFunc: func(t *testing.T) http.Handler {
				t.Helper()
				return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					hdr := getTraceIdHeader(t, req, "Other-Name")
					mustHavePrefix(t, hdr, "myorg")
					mustHaveLength(t, hdr, 41)
				})
			},
		},
		{
			name: "verbose",
			config: &Config{
				ValuePrefix: "myorg",
				HeaderName:  "Other-Name",
				Verbose:     true,
			},
			assertFunc: func(t *testing.T) http.Handler {
				t.Helper()
				return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			handler, err := New(ctx, tt.assertFunc(t), tt.config, "trace-id-test")
			if err != nil {
				t.Fatalf("error creating new plugin instance: %+v", err)
			}
			recorder := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/", nil)
			if err != nil {
				t.Fatalf("error with request: %+v", err)
			}

			handler.ServeHTTP(recorder, req)
		})
	}
}

func getTraceIdHeader(t *testing.T, req *http.Request, headerName string) string {
	t.Helper()
	headerArr := req.Header[headerName]
	if len(headerArr) == 1 {
		return headerArr[0]
	}
	return ""
}

func mustHaveLength(t *testing.T, s string, l int) {
	t.Helper()
	if len(s) != l {
		t.Fatalf("differing lengths: wanted %d, got %d(%s)", l, len(s), s)
	}
}

func mustHavePrefix(t *testing.T, s, pref string) {
	t.Helper()
	if !strings.HasPrefix(s, pref) {
		t.Fatalf("did not find prefix '%s' in '%s'(%d)", pref, s, len(s))
	}
}
