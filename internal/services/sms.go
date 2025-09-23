package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type SMSService struct {
	username string
	apiKey   string
	senderId string
	baseUrl  string
}

type SMSResponse struct {
	SMSMessageData struct {
		Message    string `json:"Message"`
		Recipients []struct {
			StatusCode int    `json:"statusCode"`
			Number     string `json:"number"`
			Status     string `json:"status"`
			Cost       string `json:"cost"`
			MessageId  string `json:"messageId"`
		} `json:"Recipients"`
	} `json:"SMSMessageData"`
}

func NewSMSService(username, apiKey, senderID string) *SMSService {
	return &SMSService{
		username: username,
		apiKey:   apiKey,
		senderId: senderID,
		baseUrl:  "https://api.sandbox.africastalking.com/version1/messaging",
	}
}

func (s *SMSService) SendSMS(to, message string) error {
	data := url.Values{}
	data.Set("username", s.username)
	data.Set("to", s.formatPhoneNumber(to))
	data.Set("message", message)
	if s.senderId != "" {
		data.Set("from", s.senderId)
	}

	req, err := http.NewRequest("POST", s.baseUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("apikey", s.apiKey) // ✅ lowercase per AT docs

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	log.Printf("SMS API response: %s", string(bodyBytes))

	var smsResponse SMSResponse
	if err := json.Unmarshal(bodyBytes, &smsResponse); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(smsResponse.SMSMessageData.Recipients) == 0 {
		return fmt.Errorf("no recipients in response")
	}

	recipient := smsResponse.SMSMessageData.Recipients[0]
	if recipient.StatusCode != 101 && recipient.StatusCode != 102 {
		return fmt.Errorf("SMS failed to send: %s (code: %d)", recipient.Status, recipient.StatusCode)
	}

	return nil
}

func (s *SMSService) SendBulkSMS(recipients []string, message string) error {
	to := strings.Join(s.formatPhoneNumbers(recipients), ",")

	data := url.Values{}
	data.Set("username", s.username)
	data.Set("to", to)
	data.Set("message", message)
	if s.senderId != "" {
		data.Set("from", s.senderId)
	}

	req, err := http.NewRequest("POST", s.baseUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("apikey", s.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	log.Printf("Bulk SMS API response: %s", string(bodyBytes))

	var smsResponse SMSResponse
	if err := json.Unmarshal(bodyBytes, &smsResponse); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	successCount := 0
	for _, recipient := range smsResponse.SMSMessageData.Recipients {
		if recipient.StatusCode == 101 || recipient.StatusCode == 102 {
			successCount++
		}
	}

	if successCount == 0 {
		return fmt.Errorf("failed to send sms to any recipient")
	}
	return nil
}

func (s *SMSService) formatPhoneNumber(phone string) string {
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")

	if strings.HasPrefix(phone, "0") {
		phone = "+254" + phone[1:]
	}
	if !strings.HasPrefix(phone, "+") {
		phone = "+254" + phone
	}
	return phone
}

func (s *SMSService) formatPhoneNumbers(phones []string) []string {
	formatted := make([]string, len(phones))
	for i, phone := range phones {
		formatted[i] = s.formatPhoneNumber(phone)
	}
	return formatted
}

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
	m.SentMessages = append(m.SentMessages, MockSMSMessage{To: to, Message: message})
	return nil
}

func (m *MockSMSService) SendBulkSMS(recipients []string, message string) error {
	for _, recipient := range recipients {
		m.SentMessages = append(m.SentMessages, MockSMSMessage{To: recipient, Message: message})
	}
	return nil
}
