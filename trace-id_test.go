package traefik_add_trace_id_header_2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateTraceID(t *testing.T) {
	testMe := &TraceIDHeader{}

	testMe.uuidGen = "4"
	got, err := testMe.GenerateTraceID()
	if err != nil || len(got) != 36 {
		t.Fatalf("Failed to return a valid UUIDv4 trace ID: %q, %v", got, err)
	}

	testMe.uuidGen = "7"
	got, err = testMe.GenerateTraceID()
	if err != nil || len(got) != 36 {
		t.Fatalf("Failed to return a valid UUIDv7 trace ID: %q, %v", got, err)
	}

	testMe.uuidGen = "L"
	got, err = testMe.GenerateTraceID()
	if err != nil || len(got) != 26 {
		t.Fatalf("Failed to return a valid ULID trace ID: %q, %v", got, err)
	}

	testMe.uuidGen = "Z" // not valid
	got, err = testMe.GenerateTraceID()
	if err == nil || got != "" {
		t.Fatalf("Expected an error and empty trace ID for an invalid uuidGen value, got %q", got)
	}
}

func TestNewRejectsInvalidUUIDGen(t *testing.T) {
	_, err := New(context.Background(), http.NotFoundHandler(), &Config{UUIDGen: "Z"}, "trace-id-test")
	if err == nil {
		t.Fatal("expected an error for an invalid uuidGen value")
	}
}

func TestNewDoesNotMutateConfig(t *testing.T) {
	config := &Config{UUIDGen: "l"}
	if _, err := New(context.Background(), http.NotFoundHandler(), config, "trace-id-test"); err != nil {
		t.Fatalf("error creating new plugin instance: %+v", err)
	}
	if config.UUIDGen != "l" {
		t.Fatalf("New modified config.UUIDGen to %q", config.UUIDGen)
	}
}

func TestServeHTTP(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		assertFunc func(t *testing.T) http.Handler
		assertResp func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
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
			name:   "incoming trace id is replaced",
			config: &Config{},
			assertFunc: func(t *testing.T) http.Handler {
				t.Helper()
				return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					hdr := getTraceIdHeader(t, req, "X-Trace-Id")
					if hdr == "client-supplied" {
						t.Fatal("incoming trace ID was not replaced")
					}
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
			name: "with suffix",
			config: &Config{
				ValueSuffix: "-sfx",
			},
			assertFunc: func(t *testing.T) http.Handler {
				t.Helper()
				return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					hdr := getTraceIdHeader(t, req, "X-Trace-Id")
					if !strings.HasSuffix(hdr, "-sfx") {
						t.Fatalf("did not find suffix '-sfx' in '%s'", hdr)
					}
					mustHaveLength(t, hdr, 40)
				})
			},
		},
		{
			name: "quoted empty prefix and suffix",
			config: &Config{
				ValuePrefix: "\"\"",
				ValueSuffix: "\"\"",
			},
			assertFunc: func(t *testing.T) http.Handler {
				t.Helper()
				return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					hdr := getTraceIdHeader(t, req, "X-Trace-Id")
					mustHaveLength(t, hdr, 36)
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
			name:   "uuidv7 via ServeHTTP",
			config: &Config{UUIDGen: "7"},
			assertFunc: func(t *testing.T) http.Handler {
				t.Helper()
				return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					hdr := getTraceIdHeader(t, req, "X-Trace-Id")
					mustHaveLength(t, hdr, 36)
					if hdr[14] != '7' {
						t.Fatalf("expected a version 7 UUID, got %s", hdr)
					}
				})
			},
		},
		{
			name:   "lowercase ulid via ServeHTTP",
			config: &Config{UUIDGen: "l"},
			assertFunc: func(t *testing.T) http.Handler {
				t.Helper()
				return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					hdr := getTraceIdHeader(t, req, "X-Trace-Id")
					mustHaveLength(t, hdr, 26)
				})
			},
		},
		{
			name: "verbose with lowercase header name",
			config: &Config{
				ValuePrefix: "myorg",
				HeaderName:  "other-name",
				Verbose:     true,
			},
			assertFunc: func(t *testing.T) http.Handler {
				t.Helper()
				return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					hdr := req.Header.Get("other-name")
					mustHavePrefix(t, hdr, "myorg")
					mustHaveLength(t, hdr, 41)
				})
			},
		},
		{
			name:   "added to response",
			config: &Config{AddToResponse: true},
			assertFunc: func(t *testing.T) http.Handler {
				t.Helper()
				return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
			},
			assertResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				t.Helper()
				mustHaveLength(t, rec.Header().Get("X-Trace-Id"), 36)
			},
		},
		{
			name:   "not added to response",
			config: &Config{AddToResponse: false},
			assertFunc: func(t *testing.T) http.Handler {
				t.Helper()
				return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
			},
			assertResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				t.Helper()
				if got := rec.Header().Get("X-Trace-Id"); got != "" {
					t.Fatalf("expected no response header, got %s", got)
				}
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
			req.Header.Set("X-Trace-Id", "client-supplied")

			handler.ServeHTTP(recorder, req)

			if tt.assertResp != nil {
				tt.assertResp(t, recorder)
			}
		})
	}
}

func getTraceIdHeader(t *testing.T, req *http.Request, headerName string) string {
	t.Helper()
	headerArr := req.Header.Values(headerName)
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

func TestTrustedIPs(t *testing.T) {
	tests := []struct {
		name       string
		trustedIPs []string
		remoteAddr string
		incoming   string
		wantKept   bool
	}{
		{name: "no trusted IPs configured", remoteAddr: "10.0.0.5:1234", incoming: "upstream-id"},
		{name: "trusted single IPv4", trustedIPs: []string{"10.0.0.5"}, remoteAddr: "10.0.0.5:1234", incoming: "upstream-id", wantKept: true},
		{name: "trusted IPv4 subnet", trustedIPs: []string{"10.0.0.0/8"}, remoteAddr: "10.200.1.1:1234", incoming: "upstream-id", wantKept: true},
		{name: "outside trusted subnet", trustedIPs: []string{"10.0.0.0/8"}, remoteAddr: "192.168.1.1:1234", incoming: "upstream-id"},
		{name: "comma separated entry", trustedIPs: []string{"192.168.0.0/16, 10.0.0.5"}, remoteAddr: "10.0.0.5:1234", incoming: "upstream-id", wantKept: true},
		{name: "trusted IPv6 subnet", trustedIPs: []string{"fd00::/8"}, remoteAddr: "[fd12::1]:1234", incoming: "upstream-id", wantKept: true},
		{name: "trusted single IPv6", trustedIPs: []string{"::1"}, remoteAddr: "[::1]:1234", incoming: "upstream-id", wantKept: true},
		{name: "IPv4-mapped IPv6 peer", trustedIPs: []string{"10.0.0.0/8"}, remoteAddr: "[::ffff:10.0.0.5]:1234", incoming: "upstream-id", wantKept: true},
		{name: "trusted but no incoming header", trustedIPs: []string{"10.0.0.0/8"}, remoteAddr: "10.0.0.5:1234"},
		{name: "unparseable remote address", trustedIPs: []string{"10.0.0.0/8"}, remoteAddr: "garbage", incoming: "upstream-id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				got = req.Header.Get("X-Trace-Id")
			})
			handler, err := New(context.Background(), next, &Config{TrustedIPs: tt.trustedIPs, AddToResponse: true}, "trace-id-test")
			if err != nil {
				t.Fatalf("error creating new plugin instance: %+v", err)
			}
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.incoming != "" {
				req.Header.Set("X-Trace-Id", tt.incoming)
			}

			handler.ServeHTTP(recorder, req)

			if tt.wantKept {
				if got != tt.incoming {
					t.Fatalf("expected incoming trace ID %q to be kept, got %q", tt.incoming, got)
				}
			} else {
				mustHaveLength(t, got, 36)
			}
			if resp := recorder.Header().Get("X-Trace-Id"); resp != got {
				t.Fatalf("response header %q does not match request header %q", resp, got)
			}
		})
	}
}

func TestNewRejectsInvalidTrustedIPs(t *testing.T) {
	for _, entry := range []string{"not-an-ip", "10.0.0.0/33", "10.0.0.256"} {
		_, err := New(context.Background(), http.NotFoundHandler(), &Config{TrustedIPs: []string{entry}}, "trace-id-test")
		if err == nil {
			t.Fatalf("expected an error for trustedIPs entry %q", entry)
		}
	}
}
