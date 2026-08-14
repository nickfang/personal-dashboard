// Package notify delivers alert notifications to the user.
//
// It lives in shared rather than in forecast-collector/internal so a second
// alert source (pollen, #69) can reuse it — Go's internal rule would block
// that. The Sender seam is what keeps a second channel (SMS for severe
// alerts) a new file rather than a rewrite.
package notify

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nickfang/personal-dashboard/services/shared"
)

// Notification is one deliverable message.
//
// Title is the whole notification as far as a phone is concerned — a lock
// screen shows the subject line and little else — so it carries the severity,
// the location, and the formatted delta rather than a generic label. Body
// carries the precise record for whoever opens the mail.
type Notification struct {
	Title string
	Body  string
	Alert shared.Alert
}

// Sender delivers a notification.
type Sender interface {
	Send(ctx context.Context, n Notification) error
}

// NopSender drops notifications. It is what an unconfigured or disabled
// delivery path resolves to, so the collector keeps working in local
// development and when the kill switch is off.
type NopSender struct{}

func (NopSender) Send(context.Context, Notification) error { return nil }

// FromAlert renders an alert as a notification.
//
// The title stays ASCII: a non-ASCII subject would need RFC 2047 encoded-word
// wrapping, and Alert.Message is already pure ASCII (see the forecast
// collector's buildMessage), so honoring this costs nothing.
func FromAlert(a shared.Alert) Notification {
	title := fmt.Sprintf("%s (%s) - %s: %s", ruleTitle(a.RuleID), a.Severity, a.Location, a.Message)

	var b strings.Builder
	fmt.Fprintf(&b, "Location:  %s\n", a.Location)
	fmt.Fprintf(&b, "Rule:      %s\n", a.RuleID)
	fmt.Fprintf(&b, "Severity:  %s\n", a.Severity)
	fmt.Fprintf(&b, "Value:     %+.1f mb (threshold %.1f)\n", a.Value, a.Threshold)
	fmt.Fprintf(&b, "Window:    %s to %s\n", formatUTC(a.WindowStart), formatUTC(a.WindowEnd))
	fmt.Fprintf(&b, "Issued at: %s\n", formatUTC(a.IssuedAt))

	return Notification{Title: title, Body: b.String(), Alert: a}
}

// ruleTitle turns a rule ID into a human label. Alert is source-agnostic, so
// this is a lookup rather than a fixed string; an unknown rule falls back to
// its own ID, which is still readable in a subject line.
func ruleTitle(ruleID string) string {
	switch {
	case strings.HasPrefix(ruleID, "pressure-drop"):
		return "Pressure drop"
	default:
		return ruleID
	}
}

// formatUTC renders a timestamp for the body. The body is the precise record;
// the local-time anchor a reader acts on is already in the title, carried
// over from Alert.Message, so this does not duplicate the collector's
// display-zone handling.
func formatUTC(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.UTC().Format("2006-01-02 15:04 MST")
}
