package notify

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

const (
	// The submission endpoint is a constant, not configuration: there is no
	// second SMTP server to point at, and a wrong value here would fail at
	// delivery rather than at startup.
	smtpHost = "smtp.gmail.com"
	smtpPort = "587"

	// dialTimeout bounds the connect; sendTimeout bounds the whole exchange.
	dialTimeout = 10 * time.Second
	sendTimeout = 30 * time.Second
)

// SMTPSender delivers notifications as email over Gmail's submission port,
// authenticating with a Google app password.
//
// It deliberately does not use smtp.SendMail. net/smtp is frozen and
// context-unaware: SendMail takes no context, sets no deadline, and dials
// bare, so a hung connection would block the job until Cloud Run's task
// timeout killed it and the ctx on Sender would be decorative. The client is
// built explicitly instead, with the connection deadline derived from ctx.
type SMTPSender struct {
	// From is the Gmail address; it doubles as the SMTP username.
	From string
	// Password is a Google app password. Gmail SMTP rejects ordinary account
	// passwords, which is why 2-Step Verification is a prerequisite.
	Password string
	// To is the recipient address.
	To string

	// Dial connects to the submission host. Nil means net.Dialer; tests
	// substitute a fake listener.
	Dial func(ctx context.Context, addr string) (net.Conn, error)

	// Now supplies the Date header. Nil means time.Now.
	Now func() time.Time
}

// NewSMTPSender builds a sender for the given credentials.
func NewSMTPSender(from, password, to string) *SMTPSender {
	return &SMTPSender{From: from, Password: password, To: to}
}

func dialTCP(ctx context.Context, addr string) (net.Conn, error) {
	d := net.Dialer{Timeout: dialTimeout}
	return d.DialContext(ctx, "tcp", addr)
}

// Send delivers one notification. Any error leaves the alert unmarked, so the
// next scheduled run retries it — there is no in-process retry.
func (s *SMTPSender) Send(ctx context.Context, n Notification) error {
	if err := s.send(ctx, n); err != nil {
		return &redactedError{err: err, secret: s.Password}
	}
	return nil
}

func (s *SMTPSender) send(ctx context.Context, n Notification) error {
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	dial := s.Dial
	if dial == nil {
		dial = dialTCP
	}
	addr := net.JoinHostPort(smtpHost, smtpPort)
	conn, err := dial(ctx, addr)
	if err != nil {
		return fmt.Errorf("dialing %s: %w", addr, err)
	}
	defer conn.Close()

	// net/smtp does no context handling of its own, so one deadline on the
	// connection is what bounds every read and write below.
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return fmt.Errorf("setting connection deadline: %w", err)
		}
	}

	client, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		return fmt.Errorf("smtp handshake with %s: %w", smtpHost, err)
	}
	defer client.Close()

	if err := client.StartTLS(&tls.Config{ServerName: smtpHost}); err != nil {
		return fmt.Errorf("starttls: %w", err)
	}
	// PlainAuth refuses to hand credentials to a non-localhost host over an
	// unencrypted connection, so the StartTLS above is enforced rather than
	// merely conventional.
	if err := client.Auth(smtp.PlainAuth("", s.From, s.Password, smtpHost)); err != nil {
		return fmt.Errorf("smtp auth as %s: %w", s.From, err)
	}

	if err := client.Mail(s.From); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err := client.Rcpt(s.To); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := w.Write(buildMessage(s.From, s.To, n, s.now())); err != nil {
		w.Close()
		return fmt.Errorf("writing message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("closing message: %w", err)
	}
	return client.Quit()
}

func (s *SMTPSender) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// buildMessage composes the RFC 5322 message. SMTP requires CRLF line
// endings and net/smtp's data writer does not convert them, so the body is
// normalized here.
func buildMessage(from, to string, n Notification, date time.Time) []byte {
	var b strings.Builder
	b.WriteString("From: " + headerValue(from) + "\r\n")
	b.WriteString("To: " + headerValue(to) + "\r\n")
	b.WriteString("Subject: " + headerValue(n.Title) + "\r\n")
	b.WriteString("Date: " + date.Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(toCRLF(n.Body))
	return []byte(b.String())
}

// headerValue strips CR and LF so a value carrying a newline cannot inject
// extra headers.
func headerValue(v string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(v)
}

func toCRLF(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\n", "\r\n")
}

// redactedError hides the app password from anything that formats the error.
// Logs are the only consumer here, and a credential written to a log outlives
// the run that leaked it.
type redactedError struct {
	err    error
	secret string
}

func (e *redactedError) Error() string {
	if e.secret == "" {
		return e.err.Error()
	}
	return strings.ReplaceAll(e.err.Error(), e.secret, "[redacted]")
}

func (e *redactedError) Unwrap() error { return e.err }
