// Package mail sends outbound email: SMTP submission when a host is
// configured, a log-only sender otherwise.
package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

// Message is one plain-text email.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Sender delivers messages. Send may block on the network; handlers call it
// in the request path, so implementations must time out on their own.
type Sender interface {
	Send(ctx context.Context, m Message) error
}

// Settings is the subset of configuration the SMTP sender needs.
type Settings struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	// TLS is "starttls", "tls" or "none".
	TLS string
}

// New returns an SMTP sender when settings name a host and a log sender
// otherwise, so callers never have to branch on configuration.
func New(s Settings) Sender {
	if strings.TrimSpace(s.Host) == "" {
		return LogSender{}
	}
	if s.Port == 0 {
		s.Port = 587
	}
	if s.TLS == "" {
		s.TLS = "starttls"
	}
	if s.From == "" {
		s.From = "no-reply@" + s.Host
	}
	return &SMTPSender{settings: s}
}

// LogSender writes the message to the process log. It exists so an instance
// without SMTP can still complete a verification or password reset by
// copying the link out of the log.
type LogSender struct{}

// Send logs the message in the same shape an operator needs to act on it.
func (LogSender) Send(_ context.Context, m Message) error {
	log.Printf("mail: no SMTP host configured, printing instead")
	log.Printf("mail: to=%s subject=%q\n%s", m.To, m.Subject, m.Body)
	return nil
}

// SMTPSender submits messages through an SMTP server.
type SMTPSender struct {
	settings Settings
}

// Send delivers one message. STARTTLS upgrades a plaintext connection;
// "tls" dials with TLS from the first byte; "none" exists for local relays
// and tests.
func (p *SMTPSender) Send(ctx context.Context, m Message) error {
	s := p.settings
	// RCPT TO and MAIL FROM go onto the wire as bare commands; an address
	// carrying CR/LF would let extra SMTP commands be injected. The message
	// body is sanitized separately in buildMessage.
	if err := validateAddr(s.From, "from"); err != nil {
		return err
	}
	if err := validateAddr(m.To, "to"); err != nil {
		return err
	}
	addr := net.JoinHostPort(s.Host, fmt.Sprint(s.Port))
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	var conn net.Conn
	var err error
	if s.TLS == "tls" {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: s.Host})
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("mail: dial %s: %w", addr, err)
	}

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	}

	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("mail: handshake: %w", err)
	}
	defer func() { _ = client.Close() }()

	if s.TLS == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: s.Host}); err != nil {
				return fmt.Errorf("mail: starttls: %w", err)
			}
		}
	}
	if s.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.Username, s.Password, s.Host)); err != nil {
			return fmt.Errorf("mail: auth: %w", err)
		}
	}
	if err := client.Mail(s.From); err != nil {
		return fmt.Errorf("mail: mail from: %w", err)
	}
	if err := client.Rcpt(m.To); err != nil {
		return fmt.Errorf("mail: rcpt to: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail: data: %w", err)
	}
	if _, err := w.Write(buildMessage(s.From, m)); err != nil {
		return fmt.Errorf("mail: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mail: close data: %w", err)
	}
	return client.Quit()
}

// buildMessage renders RFC 5322 headers plus the body. Header values are
// stripped of CR/LF so a display name or subject cannot inject headers.
func buildMessage(from string, m Message) []byte {
	var b strings.Builder
	b.WriteString("From: " + sanitizeHeader(from) + "\r\n")
	b.WriteString("To: " + sanitizeHeader(m.To) + "\r\n")
	b.WriteString("Subject: " + sanitizeHeader(m.Subject) + "\r\n")
	b.WriteString("Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("\r\n")
	// SMTP wants CRLF line endings and a terminating dot handled by the
	// client, so only normalize the newlines here.
	b.WriteString(strings.ReplaceAll(m.Body, "\n", "\r\n"))
	return []byte(b.String())
}

func sanitizeHeader(v string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(v)
}

// validateAddr refuses an envelope address that is not a single RFC 5322
// mailbox. CR/LF and other control characters would inject SMTP commands
// through MAIL FROM / RCPT TO.
func validateAddr(addr, field string) error {
	if strings.TrimSpace(addr) == "" {
		return fmt.Errorf("mail: %s address is empty", field)
	}
	parsed, err := mail.ParseAddress(addr)
	if err != nil || parsed.Address == "" {
		return fmt.Errorf("mail: %s address %q is not a valid mailbox", field, addr)
	}
	for _, r := range addr {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("mail: %s address contains a control character", field)
		}
	}
	return nil
}
