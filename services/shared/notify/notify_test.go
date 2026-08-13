package notify

import (
	"strings"
	"testing"
	"time"

	"github.com/nickfang/personal-dashboard/services/shared"
)

func testAlert() shared.Alert {
	return shared.Alert{
		ID:          "abc123",
		Location:    "house-nick",
		RuleID:      "pressure-drop-3h",
		Severity:    shared.AlertSeveritySevere,
		Value:       -6.2,
		Threshold:   5,
		WindowStart: time.Date(2026, 6, 12, 19, 0, 0, 0, time.UTC),
		WindowEnd:   time.Date(2026, 6, 12, 22, 0, 0, 0, time.UTC),
		Message:     "Thu 2 PM  -6.2 mb/3h  -8.1/6h",
		Status:      shared.AlertStatusActive,
		IssuedAt:    time.Date(2026, 6, 12, 6, 0, 0, 0, time.UTC),
	}
}

func TestFromAlert_TitleCarriesTheNotification(t *testing.T) {
	// The subject is all a phone lock screen shows, so it has to be
	// actionable on its own.
	got := FromAlert(testAlert()).Title
	want := "Pressure drop (severe) - house-nick: Thu 2 PM  -6.2 mb/3h  -8.1/6h"
	if got != want {
		t.Errorf("Title = %q, want %q", got, want)
	}
}

func TestFromAlert_TitleIsASCII(t *testing.T) {
	// A non-ASCII subject would need RFC 2047 encoded-word wrapping, which
	// buildMessage does not do.
	title := FromAlert(testAlert()).Title
	for i, r := range title {
		if r > 127 {
			t.Fatalf("Title has non-ASCII %q at %d: %q", r, i, title)
		}
	}
}

func TestFromAlert_UnknownRuleFallsBackToItsID(t *testing.T) {
	a := testAlert()
	a.RuleID = "pollen-spike-1d"
	if got := FromAlert(a).Title; !strings.HasPrefix(got, "pollen-spike-1d (severe)") {
		t.Errorf("Title = %q, want it to lead with the raw rule ID", got)
	}
}

func TestFromAlert_BodyCarriesTheFullRecord(t *testing.T) {
	body := FromAlert(testAlert()).Body
	for _, want := range []string{
		"house-nick",
		"pressure-drop-3h",
		"severe",
		"-6.2 mb (threshold 5.0)",
		"2026-06-12 19:00 UTC to 2026-06-12 22:00 UTC",
		"2026-06-12 06:00 UTC",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("Body missing %q:\n%s", want, body)
		}
	}
}

func TestBuildMessage_HeadersAndSeparator(t *testing.T) {
	date := time.Date(2026, 6, 12, 6, 0, 0, 0, time.UTC)
	n := FromAlert(testAlert())
	raw := string(buildMessage("me@gmail.com", "you@gmail.com", n, date))

	headers, body, found := strings.Cut(raw, "\r\n\r\n")
	if !found {
		t.Fatalf("no header/body separator in:\n%q", raw)
	}
	for _, want := range []string{
		"From: me@gmail.com",
		"To: you@gmail.com",
		"Subject: " + n.Title,
		"Date: Fri, 12 Jun 2026 06:00:00 +0000",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
	} {
		if !strings.Contains(headers, want) {
			t.Errorf("headers missing %q:\n%s", want, headers)
		}
	}
	if !strings.Contains(body, "Location:  house-nick") {
		t.Errorf("body missing the record:\n%s", body)
	}
	// SMTP wants CRLF and net/smtp's data writer does not convert.
	if strings.Contains(strings.ReplaceAll(raw, "\r\n", ""), "\n") {
		t.Errorf("message has bare LF line endings:\n%q", raw)
	}
}

func TestBuildMessage_SubjectCannotInjectHeaders(t *testing.T) {
	n := Notification{Title: "drop\r\nBcc: someone@example.com", Body: "body\n"}
	raw := string(buildMessage("me@gmail.com", "you@gmail.com", n, time.Now()))

	headers, _, _ := strings.Cut(raw, "\r\n\r\n")
	for _, line := range strings.Split(headers, "\r\n") {
		if strings.HasPrefix(line, "Bcc:") {
			t.Errorf("newline in the subject injected a header:\n%s", headers)
		}
	}
}

func TestNopSender_Succeeds(t *testing.T) {
	if err := (NopSender{}).Send(t.Context(), FromAlert(testAlert())); err != nil {
		t.Errorf("NopSender.Send() = %v, want nil", err)
	}
}
