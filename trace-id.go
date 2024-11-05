package traefik_add_trace_id_header_2

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/cdwiegand/traefik-add-trace-id-header-2/ulid"
	"github.com/cdwiegand/traefik-add-trace-id-header-2/uuid"
)

const defaultHeaderName = "X-Trace-Id"

// Config the plugin configuration.
type Config struct {
	ValuePrefix   string `json:"valuePrefix"`
	ValueSuffix   string `json:"valueSuffix"`
	HeaderName    string `json:"headerName"`
	Verbose       bool   `json:"verbose"`
	UuidGen       string `json:"uuidGen"`
	AddToResponse bool   `json:"addToResponse"`
}

// CreateConfig creates the DEFAULT plugin configuration - no access to config yet!
func CreateConfig() *Config {
	return &Config{
		ValuePrefix:   "",
		ValueSuffix:   "",
		HeaderName:    defaultHeaderName,
		Verbose:       false,
		UuidGen:       "4", // 4 = UUIDv4, 7 = UUIDv7, L = ULID
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
	name          string
	next          http.Handler
}

// New created a new TraceIDHeader plugin, with a config that's been set (possibly) by the admin
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config == nil {
		return nil, fmt.Errorf("config can not be nil")
	}
	if config.UuidGen == "" {
		config.UuidGen = "4" // sane default
	}
	config.UuidGen = strings.ToUpper(config.UuidGen)
	if config.UuidGen != "4" && config.UuidGen != "7" && config.UuidGen != "L" {
		return nil, fmt.Errorf("only uuid gen value of 4 (UUIDv4), 7 (UUIDv7), or L (ULID) is supported")
	}

	tIDHdr := &TraceIDHeader{
		valuePrefix:   config.ValuePrefix,
		valueSuffix:   config.ValueSuffix,
		headerName:    config.HeaderName,
		verbose:       config.Verbose,
		uuidGen:       config.UuidGen,
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

	return tIDHdr, nil
}

func (t *TraceIDHeader) GenerateTraceId() string {
	var traceValue string
	switch t.uuidGen {
	case "4":
		tmpUuid4, _ := uuid.NewV4()
		traceValue = t.valuePrefix + tmpUuid4.String()
	case "7":
		tmpUuid7, _ := uuid.NewV7()
		traceValue = t.valuePrefix + tmpUuid7.String()
	case "L":
		s2 := ulid.Make()
		traceValue = t.valuePrefix + s2.String()
	}

	return traceValue
}

func (t *TraceIDHeader) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	traceValue := t.GenerateTraceId()
	req.Header.Set(t.headerName, traceValue)
	if t.addToResponse {
		rw.Header().Set(t.headerName, traceValue)
	}

	if t.verbose {
		log.Println(t.headerName + ": " + req.Header[t.headerName][0])
	}

	t.next.ServeHTTP(rw, req)
}
