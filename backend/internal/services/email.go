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
}

// NewSMTPEmailSender sends from the Mailu mailbox configured in deployment.
// Mailu uses implicit TLS on port 465.
func NewSMTPEmailSender(host string, port int, username, password, from string) *SMTPEmailSender {
	return &SMTPEmailSender{host: host, port: port, username: username, password: password, from: from}
}

func (s *SMTPEmailSender) Send(to, subject, html string) error {
	if s.host == "" || s.port <= 0 || s.username == "" || s.password == "" || s.from == "" {
		return fmt.Errorf("Mailu SMTP is not configured")
	}
	auth := smtp.PlainAuth("", s.username, s.password, s.host)
	message := []byte("From: " + s.from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n\r\n" + html)
	address := net.JoinHostPort(s.host, strconv.Itoa(s.port))
	connection, err := tls.Dial("tcp", address, &tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12})
	if err != nil {
		return err
	}
	defer connection.Close()
	client, err := smtp.NewClient(connection, s.host)
	if err != nil {
		return err
	}
	defer client.Quit()
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
