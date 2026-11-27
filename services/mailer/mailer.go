// Package mailer composes application email messages and delegates SMTP to modules/mail.
package mailer

import "github.com/datvtph41107/bdspro/modules/mail"

// ErrUnavailable preserves the transport error contract at the service boundary.
var ErrUnavailable = mail.ErrUnavailable
