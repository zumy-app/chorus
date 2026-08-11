package services

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
)

type EmailSender interface {
	Send(to, subject, html string) error
}

type SMTPEmailSender struct {
	host     string
	port     int
	username string
	password string
	from     string
	fromName string
}

// NewSMTPEmailSender sends from the Mailu mailbox configured in deployment.
// Mailu uses implicit TLS on port 465 and STARTTLS on port 587. If from is
// empty it falls back to the username. fromName becomes the display name shown
// to recipients (defaults to "Chorus").
func NewSMTPEmailSender(host string, port int, username, password, from, fromName string) *SMTPEmailSender {
	if from == "" {
		from = username
	}
	if fromName == "" {
		fromName = "Chorus"
	}
	return &SMTPEmailSender{host: host, port: port, username: username, password: password, from: from, fromName: fromName}
}

func (s *SMTPEmailSender) Send(to, subject, html string) error {
	if s.host == "" || s.port <= 0 || s.username == "" || s.password == "" || s.from == "" {
		return fmt.Errorf("Mailu SMTP is not configured")
	}
	auth := smtp.PlainAuth("", s.username, s.password, s.host)
	fromHeader := s.from
	if s.fromName != "" {
		fromHeader = s.fromName + " <" + s.from + ">"
	}
	message := []byte("From: " + fromHeader + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n\r\n" + html)
	address := net.JoinHostPort(s.host, strconv.Itoa(s.port))

	var connection net.Conn
	var err error
	if s.port == 465 {
		// Implicit TLS (SSL).
		connection, err = tls.Dial("tcp", address, &tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12})
	} else {
		// Plain connection, then upgrade with STARTTLS (e.g. port 587).
		connection, err = net.Dial("tcp", address)
	}
	if err != nil {
		return err
	}
	defer connection.Close()
	client, err := smtp.NewClient(connection, s.host)
	if err != nil {
		return err
	}
	defer client.Quit()
	if s.port != 465 {
		if err := client.StartTLS(&tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(s.from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		return err
	}
	return writer.Close()
}
