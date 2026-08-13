package notify

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

// silentPeer returns one end of an in-memory connection whose peer accepts the
// connection and then says nothing — a stand-in for an SMTP server that never
// sends its greeting.
func silentPeer(t *testing.T) net.Conn {
	t.Helper()
	client, server := net.Pipe()
	t.Cleanup(func() {
		client.Close()
		server.Close()
	})
	return client
}

func TestSend_ContextDeadlineBoundsAHungServer(t *testing.T) {
	// Without a deadline derived from ctx this blocks until Cloud Run kills
	// the task, which is the whole reason smtp.SendMail is not used.
	s := &SMTPSender{
		From: "me@gmail.com", Password: "app-password", To: "you@gmail.com",
		Dial: func(ctx context.Context, addr string) (net.Conn, error) {
			return silentPeer(t), nil
		},
	}

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- s.Send(ctx, FromAlert(testAlert())) }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Send() = nil, want a timeout error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Send() hung past the context deadline")
	}
}

func TestSend_DialThatNeverCompletesReturns(t *testing.T) {
	// Pins that ctx actually reaches the dialer; a dial against
	// context.Background() would hang here instead.
	s := &SMTPSender{
		From: "me@gmail.com", Password: "app-password", To: "you@gmail.com",
		Dial: func(ctx context.Context, addr string) (net.Conn, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- s.Send(ctx, FromAlert(testAlert())) }()

	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Send() = %v, want a context deadline error", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Send() hung on a dial that never completes")
	}
}

func TestSend_DialsTheSubmissionEndpoint(t *testing.T) {
	var got string
	s := &SMTPSender{
		From: "me@gmail.com", Password: "app-password", To: "you@gmail.com",
		Dial: func(ctx context.Context, addr string) (net.Conn, error) {
			got = addr
			return nil, errors.New("refused")
		},
	}

	if err := s.Send(t.Context(), FromAlert(testAlert())); err == nil {
		t.Fatal("Send() = nil, want the dial error")
	}
	if want := "smtp.gmail.com:587"; got != want {
		t.Errorf("dialed %q, want %q", got, want)
	}
}

func TestSend_ErrorNeverCarriesTheAppPassword(t *testing.T) {
	const password = "abcd efgh ijkl mnop"
	s := &SMTPSender{
		From: "me@gmail.com", Password: password, To: "you@gmail.com",
		Dial: func(ctx context.Context, addr string) (net.Conn, error) {
			// Stand in for anything downstream that echoes the credential.
			return nil, errors.New("535 rejected: " + password)
		},
	}

	err := s.Send(t.Context(), FromAlert(testAlert()))
	if err == nil {
		t.Fatal("Send() = nil, want an error")
	}
	if strings.Contains(err.Error(), password) {
		t.Errorf("error leaks the app password: %q", err)
	}
	if !strings.Contains(err.Error(), "[redacted]") {
		t.Errorf("error = %q, want the credential replaced", err)
	}
}
