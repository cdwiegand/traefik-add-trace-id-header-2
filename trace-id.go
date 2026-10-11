package traefik_add_trace_id_header_2

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/cdwiegand/traefik-add-trace-id-header-2/ulid"
	"github.com/cdwiegand/traefik-add-trace-id-header-2/uuid"
)

const defaultHeaderName = "X-Trace-Id"

// Config the plugin configuration.
type Config struct {
	ValuePrefix   string   `json:"valuePrefix"`
	ValueSuffix   string   `json:"valueSuffix"`
	HeaderName    string   `json:"headerName"`
	Verbose       bool     `json:"verbose"`
	UUIDGen       string   `json:"uuidGen"`
	AddToResponse bool     `json:"addToResponse"`
	TrustedIPs    []string `json:"trustedIPs"`
}

// CreateConfig creates the DEFAULT plugin configuration - no access to config yet!
func CreateConfig() *Config {
	return &Config{
		ValuePrefix:   "",
		ValueSuffix:   "",
		HeaderName:    defaultHeaderName,
		Verbose:       false,
		UUIDGen:       "4", // 4 = UUIDv4, 7 = UUIDv7, L = ULID
		AddToResponse: true,
	}
}

// TraceIDHeader header
type TraceIDHeader struct {
	valuePrefix   string
	valueSuffix   string
	headerName    string
	verbose       bool
	uuidGen       string
	addToResponse bool
	trustedNets   []*net.IPNet
	name          string
	next          http.Handler
}

// New created a new TraceIDHeader plugin, with a config that's been set (possibly) by the admin
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config == nil {
		return nil, fmt.Errorf("config can not be nil")
	}
	uuidGen := strings.ToUpper(config.UUIDGen)
	if uuidGen == "" {
		uuidGen = "4" // sane default
	}
	if uuidGen != "4" && uuidGen != "7" && uuidGen != "L" {
		return nil, fmt.Errorf("only uuid gen value of 4 (UUIDv4), 7 (UUIDv7), or L (ULID) is supported")
	}

	tIDHdr := &TraceIDHeader{
		valuePrefix:   config.ValuePrefix,
		valueSuffix:   config.ValueSuffix,
		headerName:    config.HeaderName,
		verbose:       config.Verbose,
		uuidGen:       uuidGen,
		addToResponse: config.AddToResponse,
		next:          next,
		name:          name,
	}
	if tIDHdr.headerName == "" {
		tIDHdr.headerName = defaultHeaderName
	}
	if tIDHdr.valuePrefix == "\"\"" {
		tIDHdr.valuePrefix = "" // means use literally typed valuePrefix: "" so interpret that as empty string, not 2 double quotes (")
	}
	if tIDHdr.valueSuffix == "\"\"" {
		tIDHdr.valueSuffix = "" // means use literally typed valueSuffix: "" so interpret that as empty string, not 2 double quotes (")
	}
	trustedNets, err := parseTrustedIPs(config.TrustedIPs)
	if err != nil {
		return nil, err
	}
	tIDHdr.trustedNets = trustedNets

	return tIDHdr, nil
}

// GenerateTraceID returns a new trace ID wrapped in the configured prefix and suffix.
func (t *TraceIDHeader) GenerateTraceID() (string, error) {
	var id string
	switch t.uuidGen {
	case "4":
		u, err := uuid.NewV4()
		if err != nil {
			return "", fmt.Errorf("generating UUIDv4: %w", err)
		}
		id = u.String()
	case "7":
		u, err := uuid.NewV7()
		if err != nil {
			return "", fmt.Errorf("generating UUIDv7: %w", err)
		}
		id = u.String()
	case "L":
		id = ulid.Make().String()
	default:
		return "", fmt.Errorf("unsupported uuid gen value %q", t.uuidGen)
	}

	return t.valuePrefix + id + t.valueSuffix, nil
}

// parseTrustedIPs turns IPs and CIDR subnets into networks. Entries may also be
// comma-separated, so a single string from labels or the CLI works too.
func parseTrustedIPs(entries []string) ([]*net.IPNet, error) {
	var nets []*net.IPNet
	for _, entry := range entries {
		for _, s := range strings.Split(entry, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if strings.Contains(s, "/") {
				_, ipNet, err := net.ParseCIDR(s)
				if err != nil {
					return nil, fmt.Errorf("invalid trustedIPs subnet %q: %w", s, err)
				}
				nets = append(nets, ipNet)
				continue
			}
			ip := net.ParseIP(s)
			if ip == nil {
				return nil, fmt.Errorf("invalid trustedIPs address %q", s)
			}
			bits := 128
			if ip4 := ip.To4(); ip4 != nil {
				ip, bits = ip4, 32
			}
			nets = append(nets, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
		}
	}
	return nets, nil
}

// isTrusted reports whether the request's direct peer (not X-Forwarded-For) is in a trusted network.
func (t *TraceIDHeader) isTrusted(req *http.Request) bool {
	if len(t.trustedNets) == 0 {
		return false
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		host = req.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, n := range t.trustedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func (t *TraceIDHeader) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	traceValue := ""
	if t.isTrusted(req) {
		traceValue = req.Header.Get(t.headerName)
	}
	if traceValue == "" {
		var err error
		traceValue, err = t.GenerateTraceID()
		if err != nil {
			log.Printf("%s: failed to generate trace ID, passing request through without %s: %v", t.name, t.headerName, err)
			t.next.ServeHTTP(rw, req)
			return
		}
	}

	req.Header.Set(t.headerName, traceValue)
	if t.addToResponse {
		rw.Header().Set(t.headerName, traceValue)
	}

	if t.verbose {
		log.Println(t.headerName + ": " + traceValue)
	}

	t.next.ServeHTTP(rw, req)
}
