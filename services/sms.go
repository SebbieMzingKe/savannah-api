package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type SMSService struct {
	username string
	apiKey string
	senderId string
	baseUrl string
}

type SMSRequest struct {
	Username string `json:"username"`
	To string `json:"to"`
	Message string `json:"message"`
	From string `json:"from"`
}

type SMSResponse struct {
	SMSMessageData struct {
		Message string `json:"Message"`
		Recipients []struct {
			StatusCode int `json:"statusCode"`
			Number string `json:"number"`
			Status string `json:"statusCstatusode"`
			Cost string `json:"cost"`
			MessageId string `json:"messageId"`
		} `json:"Recipients"`
	} `json:"SMSMessageData"`
}

func NewSMSService(username, apiKey, senderID string) *SMSService {
	return &SMSService{
		username: username,
		apiKey: apiKey,
		senderId: senderID,
		baseUrl: "https://api.sandbox.africastalking.com/version1/messaging",
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
	req.Header.Set("apiKey", s.apiKey)

	client := http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return  fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	var smsResponse SMSResponse
	if err := json.NewDecoder(resp.Body).Decode(&smsResponse); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	
	if (len(smsResponse.SMSMessageData.Recipients)) == 0 {
		return fmt.Errorf("no recipiients in response")
	}

	recipient := smsResponse.SMSMessageData.Recipients[0]
	if recipient.StatusCode != 101 && recipient.StatusCode != 102 {
		return fmt.Errorf("SMS failed to send: %s (code: %d)", recipient.Status, recipient.StatusCode)
	}
	return nil
}

func (s *SMSService) formatPhoneNumber(phone string) string {
	return phone
}