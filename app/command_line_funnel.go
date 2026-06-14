// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/types"
	"oblikovati.org/app/cmdline"
)

// The notification funnel (M26 F03): the Command Window is the single feedback surface, so
// every messaging path — the status notice, the message center, balloon tips, and prompt
// questions — appends to its rolling scrollback. The old toast / message-center / prompt
// modal windows are retired in the head; the underlying centers stay as the data sources.

// feedScrollback appends one line to the Command Window's rolling output (empty ignored).
func (s *Session) feedScrollback(text string, sev cmdline.Severity) {
	if text == "" {
		return
	}
	s.CommandLine().Scrollback().Append(text, sev)
}

// feedNotice routes a status notice into the command line (M26 F03).
func (s *Session) feedNotice(msg string) { s.feedScrollback(msg, cmdline.Info) }

// routeMessage forwards a message-center entry to the command line, mapping its severity —
// the MessageCenter sink wired in NewSession.
func (s *Session) routeMessage(text string, severity types.MessageSeverity) {
	s.feedScrollback(text, messageSeverity(severity))
}

// messageSeverity maps a message-center severity to a command-line severity.
func messageSeverity(sev types.MessageSeverity) cmdline.Severity {
	switch sev {
	case types.SeverityError:
		return cmdline.Error
	case types.SeverityWarning:
		return cmdline.Warning
	default:
		return cmdline.Info
	}
}

// balloonLine formats a balloon tip as one command-line line.
func balloonLine(spec BalloonTipSpec) string {
	if spec.Text == "" {
		return spec.Title
	}
	return spec.Title + " — " + spec.Text
}
