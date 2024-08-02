package traceinjector

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	uuid "github.com/gofrs/uuid/v5"
)

const defaultHeaderName = "X-Trace-Id"

// Config the plugin configuration.
type Config struct {
	ValuePrefix     string `json:"valuePrefix"`
	ValueSuffix     string `json:"valueSuffix"`
	HeaderName      string `json:"headerName"`
	Verbose         bool   `json:"verbose"`
	TrustAllIPs     bool   `json:"trustAllIPs"`
	TrustPrivateIPs bool   `json:"trustPrivateIPs"`
	TrustLocalhost  bool   `json:"trustLocalhost"`
	TrustNetworks   string `json:"trustNetworks"`
	UuidGen         int    `json:"uuidGen"`
}

// CreateConfig creates the DEFAULT plugin configuration - no access to config yet!
func CreateConfig() *Config {
	return &Config{
		ValuePrefix:     "",
		ValueSuffix:     "",
		HeaderName:      defaultHeaderName,
		TrustNetworks:   "",
		TrustAllIPs:     false,
		TrustPrivateIPs: false,
		TrustLocalhost:  false,
		Verbose:         false,
		UuidGen:         4,
	}
}

// TraceIDHeader header
type TraceIDHeader struct {
	valuePrefix     string
	valueSuffix     string
	headerName      string
	verbose         bool
	trustNetworks   []*net.IPNet
	trustAllIPs     bool
	trustPrivateIPs bool
	trustLocalhost  bool
	uuidGen         int
	name            string
	next            http.Handler
}

// New created a new TraceIDHeader plugin, with a config that's been set (possibly) by the admin
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config == nil {
		return nil, fmt.Errorf("config can not be nil")
	}
	if config.UuidGen == 0 {
		config.UuidGen = 4 // sane default
	}
	if config.UuidGen != 4 && config.UuidGen != 7 {
		return nil, fmt.Errorf("only uuid gen value of 4 or 7 is supported")
	}

	var trustedIPRanges []*net.IPNet
	if config.TrustNetworks == "*" {
		config.TrustAllIPs = true
	} else if config.TrustNetworks != "" {
		for _, v := range strings.Split(config.TrustNetworks, ",") {
			_, ipnet, err := net.ParseCIDR(v)
			if err != nil {
				return nil, err
			}
			trustedIPRanges = append(trustedIPRanges, ipnet)
		}
	}

	tIDHdr := &TraceIDHeader{
		valuePrefix:     config.ValuePrefix,
		valueSuffix:     config.ValueSuffix,
		headerName:      config.HeaderName,
		verbose:         config.Verbose,
		trustAllIPs:     config.TrustAllIPs,
		trustPrivateIPs: config.TrustPrivateIPs,
		trustLocalhost:  config.TrustLocalhost,
		trustNetworks:   trustedIPRanges,
		uuidGen:         config.UuidGen,
		next:            next,
		name:            name,
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

	return tIDHdr, nil
}

func (t *TraceIDHeader) ModifyRequest(req *http.Request) {
	var s uuid.UUID
	switch t.uuidGen {
	case 4:
		s, _ = uuid.NewV4()
	case 7:
		s, _ = uuid.NewV7()
	}
	randomUUID := t.valuePrefix + s.String()

	req.Header.Del(t.headerName)
	req.Header.Add(t.headerName, randomUUID)

	if t.verbose {
		log.Println(req.Header[t.headerName][0])
	}
}

func (t *TraceIDHeader) IsIpTrusted(ip net.IP) bool {
	if t.trustAllIPs {
		return true
	} else if t.trustPrivateIPs && (net.IP.IsPrivate(ip) || net.IP.IsLoopback(ip)) {
		return true
	} else if t.trustLocalhost && net.IP.IsLoopback(ip) {
		return true
	} else {
		for _, v := range t.trustNetworks {
			if v.Contains(ip) {
				return true
			}
		}
	}
	return false
}

func (t *TraceIDHeader) ExtractRemoteIp(req *http.Request) net.IP {
	ip := req.RemoteAddr
	if ip == "" {
		return net.IPv4zero
	}
	if strings.Contains(ip, ":") {
		ip, _, _ = net.SplitHostPort(ip)
	}
	return net.ParseIP(ip)
}

func (t *TraceIDHeader) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	remoteIp := t.ExtractRemoteIp(req)
	if remoteIp == nil || remoteIp.IsUnspecified() || !t.IsIpTrusted(remoteIp) {
		t.ModifyRequest(req)
	}

	t.next.ServeHTTP(rw, req)
}
