package traceinjector

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsIpTrusted(t *testing.T) {
	testMe_allTrust := &TraceIDHeader{
		trustAllIPs: true,
	}
	testMe_privateTrust := &TraceIDHeader{
		trustPrivateIPs: true,
	}
	var specificNetworks []*net.IPNet
	_, cidr, _ := net.ParseCIDR("10.0.0.0/24")
	specificNetworks = append(specificNetworks, cidr)
	testMe_specificTrust := &TraceIDHeader{
		trustNetworks: specificNetworks,
	}
	testMe_trustNoOne := &TraceIDHeader{}
	testMe_trustLocalOnly := &TraceIDHeader{
		trustLocalhost: true,
	}

	ipv4Local := "127.0.0.1"
	ipv4Local2 := "127.0.0.2"
	ipv4Public := "1.2.3.4"
	ipv4PrivateGood := "10.0.0.1"
	ipv4PrivateBad := "192.168.1.1"
	ipv6Localhost := "::1"
	ipv6Public := "2006:1234:5678:9090:abcd:beef:feed:0001"
	ipv6Private := "fc00:1234:5678:9090:abcd:beef:feed:0001"

	tests := []struct {
		name       string
		testPlugin *TraceIDHeader
		ip         string
		expected   bool
	}{
		{
			name: "allTrust_ipv4_localhost", testPlugin: testMe_allTrust, ip: ipv4Local, expected: true,
		},
		{
			name: "allTrust_ipv6_localhost", testPlugin: testMe_allTrust, ip: ipv6Localhost, expected: true,
		},
		{
			name: "allTrust_ipv4_public", testPlugin: testMe_allTrust, ip: ipv4Public, expected: true,
		},
		{
			name: "allTrust_ipv6_public", testPlugin: testMe_allTrust, ip: ipv6Public, expected: true,
		},
		{
			name: "allTrust_ipv4_private", testPlugin: testMe_allTrust, ip: ipv4PrivateGood, expected: true,
		},
		{
			name: "allTrust_ipv6_private", testPlugin: testMe_allTrust, ip: ipv6Private, expected: true,
		},

		{
			name: "privateTrust_ipv4_localhost", testPlugin: testMe_privateTrust, ip: ipv4Local, expected: true,
		},
		{
			name: "privateTrust_ipv4_localhost2", testPlugin: testMe_privateTrust, ip: ipv4Local2, expected: true,
		},
		{
			name: "privateTrust_ipv6_localhost", testPlugin: testMe_privateTrust, ip: ipv6Localhost, expected: true,
		},
		{
			name: "privateTrust_ipv4_public", testPlugin: testMe_privateTrust, ip: ipv4Public, expected: false,
		},
		{
			name: "privateTrust_ipv6_public", testPlugin: testMe_privateTrust, ip: ipv6Public, expected: false,
		},
		{
			name: "privateTrust_ipv4_privateGood", testPlugin: testMe_privateTrust, ip: ipv4PrivateGood, expected: true,
		},
		{
			name: "privateTrust_ipv4_privateBad", testPlugin: testMe_privateTrust, ip: ipv4PrivateBad, expected: true,
		},
		{
			name: "privateTrust_ipv6_private", testPlugin: testMe_privateTrust, ip: ipv6Private, expected: true,
		},

		{
			name: "specificTrust_ipv4_localhost", testPlugin: testMe_specificTrust, ip: ipv4Local, expected: false,
		},
		{
			name: "specificTrust_ipv6_localhost", testPlugin: testMe_specificTrust, ip: ipv6Localhost, expected: false,
		},
		{
			name: "specificTrust_ipv4_public", testPlugin: testMe_specificTrust, ip: ipv4Public, expected: false,
		},
		{
			name: "specificTrust_ipv6_public", testPlugin: testMe_specificTrust, ip: ipv6Public, expected: false,
		},
		{
			name: "specificTrust_ipv4_privateGood", testPlugin: testMe_specificTrust, ip: ipv4PrivateGood, expected: true,
		},
		{
			name: "specificTrust_ipv4_privateBad", testPlugin: testMe_specificTrust, ip: ipv4PrivateBad, expected: false,
		},
		{
			name: "specificTrust_ipv6_private", testPlugin: testMe_specificTrust, ip: ipv6Private, expected: false,
		},

		{
			name: "trustNoOne_ipv4_localhost", testPlugin: testMe_trustNoOne, ip: ipv4Local, expected: false,
		},
		{
			name: "trustNoOne_ipv6_localhost", testPlugin: testMe_trustNoOne, ip: ipv6Localhost, expected: false,
		},
		{
			name: "trustNoOne_ipv4_public", testPlugin: testMe_trustNoOne, ip: ipv4Public, expected: false,
		},
		{
			name: "trustNoOne_ipv6_public", testPlugin: testMe_trustNoOne, ip: ipv6Public, expected: false,
		},
		{
			name: "trustNoOne_ipv4_private", testPlugin: testMe_trustNoOne, ip: ipv4PrivateGood, expected: false,
		},
		{
			name: "trustNoOne_ipv6_private", testPlugin: testMe_trustNoOne, ip: ipv6Private, expected: false,
		},

		{
			name: "trustLocalOnly_ipv4_localhost", testPlugin: testMe_trustLocalOnly, ip: ipv4Local, expected: true,
		},
		{
			name: "trustLocalOnly_ipv4_localhost2", testPlugin: testMe_trustLocalOnly, ip: ipv4Local2, expected: true,
		},
		{
			name: "trustLocalOnly_ipv6_localhost", testPlugin: testMe_trustLocalOnly, ip: ipv6Localhost, expected: true,
		},
		{
			name: "trustLocalOnly_ipv4_public", testPlugin: testMe_trustLocalOnly, ip: ipv4Public, expected: false,
		},
		{
			name: "trustLocalOnly_ipv6_public", testPlugin: testMe_trustLocalOnly, ip: ipv6Public, expected: false,
		},
		{
			name: "trustLocalOnly_ipv4_private", testPlugin: testMe_trustLocalOnly, ip: ipv4PrivateGood, expected: false,
		},
		{
			name: "trustLocalOnly_ipv6_private", testPlugin: testMe_trustLocalOnly, ip: ipv6Private, expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			got := tt.testPlugin.IsIpTrusted(ip)
			if got != tt.expected {
				t.Errorf("Got wrong boolean result when given '%s'", tt.ip)
			}
		})
	}
}

func TestExtractRemoteIp(t *testing.T) {
	testMe := &TraceIDHeader{}
	req, _ := http.NewRequest("GET", "/", nil)
	got := testMe.ExtractRemoteIp(req)
	if got.String() != "0.0.0.0" {
		t.Fatalf("Got '%s' but expected '0.0.0.0'", got.String())
	}
	req.RemoteAddr = "1.2.3.4"
	got = testMe.ExtractRemoteIp(req)
	if got.String() != "1.2.3.4" {
		t.Fatalf("Got '%s' but expected '1.2.3.4'", got.String())
	}
}

func TestModifyRequest(t *testing.T) {
	testMe := &TraceIDHeader{}
	req, _ := http.NewRequest("GET", "/", nil)
	testMe.ModifyRequest(req)
	got := req.Header[testMe.headerName]
	if len(got) != 1 || len(got[0]) < 1 {
		t.Fatal("Failed to set a valid trace header.")
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
