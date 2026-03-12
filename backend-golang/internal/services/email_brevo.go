package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type EmailAttachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

type EmailMessage struct {
	To          []string
	Subject     string
	HTMLContent string
	Attachments []EmailAttachment
}

type EmailService interface {
	SendEmail(ctx context.Context, msg EmailMessage) error
}

type EmailServiceWithID interface {
	SendEmailWithID(ctx context.Context, msg EmailMessage) (string, error)
}

type BrevoEmailService struct {
	apiKey    string
	fromEmail string
	fromName  string
	client    *http.Client
}

func NewBrevoEmailService(apiKey, fromEmail, fromName string) *BrevoEmailService {
	return &BrevoEmailService{
		apiKey:    apiKey,
		fromEmail: fromEmail,
		fromName:  fromName,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *BrevoEmailService) SendEmail(ctx context.Context, msg EmailMessage) error {
	_, err := s.SendEmailWithID(ctx, msg)
	return err
}

func (s *BrevoEmailService) SendEmailWithID(ctx context.Context, msg EmailMessage) (string, error) {
	if s.apiKey == "" || s.fromEmail == "" {
		return "", fmt.Errorf("brevo api key and from email are required")
	}
	if len(msg.To) == 0 {
		return "", fmt.Errorf("email recipients are required")
	}

	type recipient struct {
		Email string `json:"email"`
	}

	payload := map[string]interface{}{
		"sender": map[string]string{
			"email": s.fromEmail,
			"name":  s.fromName,
		},
		"to":          []recipient{},
		"subject":     msg.Subject,
		"htmlContent": msg.HTMLContent,
	}

	toList := []recipient{}
	for _, t := range msg.To {
		if t == "" {
			continue
		}
		toList = append(toList, recipient{Email: t})
	}
	payload["to"] = toList

	if len(msg.Attachments) > 0 {
		attachments := []map[string]string{}
		for _, a := range msg.Attachments {
			if len(a.Content) == 0 || a.Filename == "" {
				continue
			}
			attachments = append(attachments, map[string]string{
				"name":    a.Filename,
				"content": base64.StdEncoding.EncodeToString(a.Content),
			})
		}
		if len(attachments) > 0 {
			payload["attachment"] = attachments
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.brevo.com/v3/smtp/email", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("brevo send failed: status %d", resp.StatusCode)
	}

	messageID := ""
	if bodyBytes, err := io.ReadAll(resp.Body); err == nil && len(bodyBytes) > 0 {
		var payload struct {
			MessageID string `json:"messageId"`
		}
		if json.Unmarshal(bodyBytes, &payload) == nil {
			messageID = payload.MessageID
		}
	}
	return messageID, nil
}
