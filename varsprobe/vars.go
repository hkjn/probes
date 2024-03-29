// Package varsprobe implements a probe for /vars, i.e. expvar package variables.
package varsprobe // import "hkjn.me/probes/varsprobe"

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"hkjn.me/prober"
	"hkjn.me/probes"
)

// VarsProber probes a target host's /vars page.
type VarsProber struct {
	Target     string // addr to probe /vars for
	Key        string // variable key to probe for
	name, desc string // name and description of the prober
	wantValue  string // value for key to expect
	alertFn    prober.AlertFn
}

// Alert sets a custom alerting function.
//
// If Alert is not called, the probes.SendAlertEmail function is called.
func Alert(fn prober.AlertFn) func(*VarsProber) {
	return func(p *VarsProber) {
		p.alertFn = fn
	}
}

// New returns a new instance of the vars probe with specified options.
func New(target string, options ...func(*VarsProber)) *prober.Probe {
	return NewWithGeneric(target, []prober.Option{}, options...)
}

// NewWithGeneric returns a new instance of the vars probe with specified options.
//
// NewWithGeneric passes through specified prober.Options, after
// applying the varsprobe-specific options.
func NewWithGeneric(target string, genericOpts []prober.Option, options ...func(*VarsProber)) *prober.Probe {
	p := &VarsProber{
		Target:  target,
		alertFn: probes.SendAlertEmail,
	}
	for _, opt := range options {
		opt(p)
	}

	// Set a default name if none was specified.
	if p.name == "" {
		p.name = fmt.Sprintf("VarsProber_%s_%s_%s", target, p.Key, p.wantValue)
	}
	// Set a default desc if none was specified.
	if p.desc == "" {
		p.desc = fmt.Sprintf("Probes vars page of %s for key %s, value %s", target, p.Key, p.wantValue)
	}
	// TODO(hkjn): This currently doesn't let prober.Interval be
	// overridden, because even if it's supplied in genericOpts we
	// override it ourselves..
	return prober.NewProbe(p, p.name, p.desc,
		append(genericOpts, prober.Interval(time.Minute*5), prober.FailurePenalty(5))...)
}

// Name sets specified name.
func Name(name string) func(*VarsProber) {
	return func(p *VarsProber) {
		p.name = name
	}
}

// Key sets key to check.
func Key(k string) func(*VarsProber) {
	return func(p *VarsProber) {
		p.Key = k
	}
}

// Desc sets specified description.
func Desc(desc string) func(*VarsProber) {
	return func(p *VarsProber) {
		p.desc = desc
	}
}

// WantValue sets value to expect key to have.
func WantValue(v string) func(*VarsProber) {
	return func(p *VarsProber) {
		p.wantValue = v
	}
}

// Probe verifies that the target's /vars page is as expected.
func (p *VarsProber) Probe() prober.Result {
	if err := p.check(); err != nil {
		return prober.FailedWith(err)
	}
	return prober.PassedWith(p.String(), p.Target)
}

// String returns the human-readable description of the prober.
func (p VarsProber) String() string {
	return fmt.Sprintf("%s: %s (%s=%s)", p.name, p.desc, p.Key, p.wantValue)
}

// check verifies that the target has expected key and value on /vars page.
func (p *VarsProber) check() error {
	resp, err := http.Get(p.Target) // http://ln.hkjn.me/debug/vars
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %v", p.Target, err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}
	log.Printf("FIXMEH: %q got resp %q\n", p.Target, string(b))
	// todo: parse JSON and look up p.Key and expect p.wantValue
	return nil
}

// Alert calls the prober.AlertFn for the prober.
//
// If no prober.AlertFn was set with the Alert() option,
// probes.SendAlertEmail is used by default.
func (p *VarsProber) Alert(name, desc string, badness int, records prober.Records) error {
	return p.alertFn(name, desc, badness, records)
}
