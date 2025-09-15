package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type SMSService struct {
	username string
	apiKey   string
	senderID string
	baseURL  string
}

type SMSRequest struct {
	Username string `json:"username"`
	To       string `json:"to"`
	Message  string `json:"message"`
	From     string `json:"from,omitempty"`
}

type SMSResponse struct {
	SMSMessageData struct {
		Message    string `json:"Message"`
		Recipients []struct {
			StatusCode   int    `json:"statusCode"`
			Number       string `json:"number"`
			Status       string `json:"status"`
			Cost         string `json:"cost"`
			MessageID    string `json:"messageId"`
			MessageParts int    `json:"messageParts"`
		} `json:"Recipients"`
	} `json:"SMSMessageData"`
}

// NewSMSService creates a new SMS service instance
func NewSMSService(username, apiKey, senderID string) *SMSService {
	return &SMSService{
		username: username,
		apiKey:   apiKey,
		senderID: senderID,
		baseURL:  "https://api.sandbox.africastalking.com/version1/messaging", // Use sandbox for testing
	}
}

// SendSMS sends an SMS message using Africa's Talking API
func (s *SMSService) SendSMS(to, message string) error {
	// Prepare the request data
	data := url.Values{}
	data.Set("username", s.username)
	data.Set("to", s.formatPhoneNumber(to))
	data.Set("message", message)
	if s.senderID != "" {
		data.Set("from", s.senderID)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", s.baseURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("apiKey", s.apiKey)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse the response
	var smsResponse SMSResponse
	if err := json.NewDecoder(resp.Body).Decode(&smsResponse); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if the SMS was sent successfully
	if len(smsResponse.SMSMessageData.Recipients) == 0 {
		return fmt.Errorf("no recipients in response")
	}

	recipient := smsResponse.SMSMessageData.Recipients[0]
	if recipient.StatusCode != 101 && recipient.StatusCode != 102 {
		return fmt.Errorf("SMS failed to send: %s (code: %d)", recipient.Status, recipient.StatusCode)
	}

	return nil
}

// SendBulkSMS sends SMS to multiple recipients
func (s *SMSService) SendBulkSMS(recipients []string, message string) error {
	// Join phone numbers with commas
	to := strings.Join(s.formatPhoneNumbers(recipients), ",")

	// Prepare the request data
	data := url.Values{}
	data.Set("username", s.username)
	data.Set("to", to)
	data.Set("message", message)
	if s.senderID != "" {
		data.Set("from", s.senderID)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", s.baseURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("apiKey", s.apiKey)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse the response
	var smsResponse SMSResponse
	if err := json.NewDecoder(resp.Body).Decode(&smsResponse); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if at least one SMS was sent successfully
	successCount := 0
	for _, recipient := range smsResponse.SMSMessageData.Recipients {
		if recipient.StatusCode == 101 || recipient.StatusCode == 102 {
			successCount++
		}
	}

	if successCount == 0 {
		return fmt.Errorf("failed to send SMS to any recipient")
	}

	return nil
}

// formatPhoneNumber formats a phone number for Africa's Talking API
// Ensures the number is in international format with country code
func (s *SMSService) formatPhoneNumber(phone string) string {
	// Remove any spaces, dashes, or parentheses
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")

	// If the number starts with 0, replace with +254 (Kenya country code)
	if strings.HasPrefix(phone, "0") {
		phone = "+254" + phone[1:]
	}

	// If the number doesn't start with +, add +254
	if !strings.HasPrefix(phone, "+") {
		phone = "+254" + phone
	}

	return phone
}

// formatPhoneNumbers formats multiple phone numbers
func (s *SMSService) formatPhoneNumbers(phones []string) []string {
	formatted := make([]string, len(phones))
	for i, phone := range phones {
		formatted[i] = s.formatPhoneNumber(phone)
	}
	return formatted
}

// MockSMSService for testing purposes
type MockSMSService struct {
	SentMessages []MockSMSMessage
}

type MockSMSMessage struct {
	To      string
	Message string
}

func NewMockSMSService() *MockSMSService {
	return &MockSMSService{
		SentMessages: make([]MockSMSMessage, 0),
	}
}

func (m *MockSMSService) SendSMS(to, message string) error {
	m.SentMessages = append(m.SentMessages, MockSMSMessage{
		To:      to,
		Message: message,
	})
	return nil
}

func (m *MockSMSService) SendBulkSMS(recipients []string, message string) error {
	for _, recipient := range recipients {
		m.SentMessages = append(m.SentMessages, MockSMSMessage{
			To:      recipient,
			Message: message,
		})
	}
	return nil
}