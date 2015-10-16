// Package webprobe implements a HTTP probe.
package webprobe

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/hkjn/prober"
	"github.com/hkjn/probes"
)

const (
	MaxResponseBytes int64 = 10e5 // largest response size accepted
	defaultName            = "WebProber"
)

// WebProber probes a target's HTTP response.
type WebProber struct {
	Target         string // URL to probe
	Method         string // GET, POST, PUT, etc.
	Name           string // name of the prober
	Body           io.Reader
	wantCode       int
	wantInResponse string
}

// Name sets the name for the prober.
func Name(name string) func(*WebProber) {
	return func(p *WebProber) {
		p.Name = fmt.Sprintf("%s_%s", defaultName, name)
	}
}

// Body sets the HTTP request body for the prober.
func Body(body io.Reader) func(*WebProber) {
	return func(p *WebProber) {
		p.Body = body
	}
}

// InResponse applies the option that the prober wants given string in the HTTP response.
func InResponse(str string) func(*WebProber) {
	return func(p *WebProber) {
		p.wantInResponse = str
	}
}

// New returns a new instance of the web probe with specified options.
func New(target, method string, code int, options ...func(*WebProber)) *prober.Probe {
	return NewWithGeneric(target, method, code, []prober.Option{}, options...)
}

// NewWithGeneric returns a new instance of the web probe with specified options.
//
// NewWithGeneric passes through specified prober.Options, after
// applying the webprobe-specific options.
func NewWithGeneric(target, method string, code int, genericOpts []prober.Option, options ...func(*WebProber)) *prober.Probe {
	name := defaultName
	p := &WebProber{Target: target, Name: name, Method: method, wantCode: code}
	for _, opt := range options {
		opt(p)
	}
	return prober.NewProbe(p, p.Name, fmt.Sprintf("Probes HTTP response of %s", target), genericOpts...)
}

// Probe verifies that the target's HTTP response is as expected.
func (p WebProber) Probe() prober.Result {
	req, err := http.NewRequest(p.Method, p.Target, p.Body)
	if err != nil {
		return prober.FailedWith(fmt.Errorf("failed to create HTTP request: %v", err))
	}
	// Inform the server that we'd like the connection to be closed once
	// we're done:
	// http://craigwickesser.com/2015/01/golang-http-to-many-open-files/
	req.Header.Set("Connection", "close")
	t := http.Transport{}
	resp, err := t.RoundTrip(req)
	if err != nil {
		return prober.FailedWith(fmt.Errorf("failed to send HTTP request: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != p.wantCode {
		return prober.FailedWith(fmt.Errorf("bad HTTP response status; want %d, got %d", p.wantCode, resp.StatusCode))
	}
	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes))
	if err != nil {
		return prober.FailedWith(fmt.Errorf("failed to read HTTP response: %v", err))
	}
	sb := string(body)
	if !strings.Contains(sb, p.wantInResponse) {
		return prober.FailedWith(fmt.Errorf("response doesn't contain %q: \n%v\n", p.wantInResponse, sb))
	}
	return prober.Passed()
}

// Alert sends an alert notification via email.
func (p *WebProber) Alert(name, desc string, badness int, records prober.Records) error {
	return probes.SendAlertEmail(name, desc, badness, records)
}
