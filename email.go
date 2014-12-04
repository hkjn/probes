// Package probes provides some probe implementations.
//
// This package defines some helpers to send alert emails, while
// actual probes are defined in subpackages.
package probes // import "hkjn.me/probes"

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/golang/glog"
	"github.com/sendgrid/sendgrid-go"
	"hkjn.me/prober"
)

type ConfigT struct {
	Template *template.Template
	Alert    struct {
		// From:, To: addresses for alert emails.
		Sender, Recipient string
		CCs               []string
	}
	Sendgrid struct {
		User, Password string
	}
}

var Config = ConfigT{}

func getClient() (*sendgrid.SGClient, error) {
	user := Config.Sendgrid.User
	pw := Config.Sendgrid.Password
	if user == "" {
		return nil, fmt.Errorf("no sendgrid user specified - set Config.Sendgrid.User")
	}
	if pw == "" {
		return nil, fmt.Errorf("no sendgrid password specified - set Config.Sendgrid.Password")
	}
	return sendgrid.NewSendGridClient(user, pw), nil
}

// SendAlertEmail sends an alert email.
func SendAlertEmail(name, desc string, badness int, records prober.Records) error {
	glog.V(1).Infof("sending alert email..\n")
	data := struct {
		Name, Desc string
		Badness    int
		Records    prober.Records
	}{name, desc, badness, records}

	var html bytes.Buffer
	err := Config.Template.ExecuteTemplate(&html, "email", data)
	if err != nil {
		return fmt.Errorf("failed to construct email from template: %v", err)
	}

	m := sendgrid.NewMail()
	subject := fmt.Sprintf("%s failed (badness %d)", name, badness)
	m.SetSubject(subject)
	err = m.AddTo(Config.Alert.Recipient)
	if err != nil {
		return fmt.Errorf("failed to add recipients: %v", err)
	}
	err = m.AddCcs(Config.Alert.CCs)
	if err != nil {
		return fmt.Errorf("failed to add cc recipients: %v", err)
	}
	m.SetHTML(html.String())
	err = m.SetFrom(Config.Alert.Sender)
	if err != nil {
		return fmt.Errorf("failed to add sender %q: %v", Config.Alert.Sender, err)
	}
	sgClient, err := getClient()
	if err != nil {
		return fmt.Errorf("failed to create mail client: %v", err)
	}
	err = sgClient.Send(m)
	if err != nil {
		return fmt.Errorf("failed to send mail: %v", err)
	}
	glog.Infof("sent alert email to %s\n", Config.Alert.Recipient)
	return nil
}
