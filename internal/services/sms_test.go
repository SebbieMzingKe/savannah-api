package services

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestFormatPhoneNumber(t *testing.T) {
	smsService := NewSMSService("test", "test", "test")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "kenyan number with leading zero",
			input:    "0770110234",
			expected: "+254770110234",
		},
		{
			name:     "Kenyan number without country code",
			input:    "701234567",
			expected: "+254701234567",
		},
		{
			name:     "Number with spaces",
			input:    "0701 234 567",
			expected: "+254701234567",
		},
		{
			name:     "Number with dashes",
			input:    "0701-234-567",
			expected: "+254701234567",
		},
		{
			name:     "Number with parentheses",
			input:    "(0701)234567",
			expected: "+254701234567",
		},
		{
			name:     "Already formatted international number",
			input:    "+254701234567",
			expected: "+254701234567",
		},
		{
			name:     "Number with mixed formatting",
			input:    "0701 (234) 567",
			expected: "+254701234567",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := smsService.formatPhoneNumber(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatPhoneNumbers(t *testing.T) {
	smsService := NewSMSService("test", "test", "test")

	input := []string{
		"0740827150",
		"0740657150",
		"+254740827159",
	}

	expected := []string{
		"+254740827150",
		"+254740657150",
		"+254740827159",
	}
	result := smsService.formatPhoneNumbers(input)
	assert.Equal(t, expected, result)
}

func TestMockSMSService(t *testing.T) {
	mockService := NewMockSMSService()

	t.Run("send single sms", func(t *testing.T) {
		mockService.SentMessages = nil
		to := "+254740827150"
		message := "test message"

		err := mockService.SendSMS(to, message)
		assert.NoError(t, err)

		assert.Len(t, mockService.SentMessages, 1)
		assert.Equal(t, to, mockService.SentMessages[0].To)
		assert.Equal(t, message, mockService.SentMessages[0].Message)
	})

	t.Run("send bulk sms", func(t *testing.T) {
		mockService.SentMessages = nil
		recipients := []string{"+254740827150", "+254111768132", "+254770110234"}
		message := "bulk test message"

		err := mockService.SendBulkSMS(recipients, message)
		assert.NoError(t, err)

		assert.Len(t, mockService.SentMessages, 3)

		for i, recipient := range recipients {
			sentMessage := mockService.SentMessages[i]
			assert.Equal(t, recipient, sentMessage.To)
			assert.Equal(t, message, sentMessage.Message)
		}
	})

	t.Run("clear and send new messages", func(t *testing.T) {
		mockService.SentMessages = nil
		to := "+254740827150"
		message := "new test message"

		err := mockService.SendSMS(to, message)
		assert.NoError(t, err)

		assert.Len(t, mockService.SentMessages, 1)
		assert.Equal(t, to, mockService.SentMessages[0].To)
		assert.Equal(t, message, mockService.SentMessages[0].Message)
	})
}

func TestSMSServiceCreation(t *testing.T) {
	username := "testuser"
	apiKey := "testapikey"
	senderID := "testsender"

	smsService := NewSMSService(username, apiKey, senderID)

	assert.Equal(t, username, smsService.username)
	assert.Equal(t, apiKey, smsService.apiKey)
	assert.Equal(t, senderID, smsService.senderId)
	assert.Equal(t, "https://api.sandbox.africastalking.com/version1/messaging", smsService.baseUrl)
}

func TestSMSServiceWithEmptySenderID(t *testing.T) {
	username := "testuser"
	apiKey := "testapikey"
	senderID := ""

	smsService := NewSMSService(username, apiKey, senderID)

	assert.Equal(t, username, smsService.username)
	assert.Equal(t, apiKey, smsService.apiKey)
	assert.Equal(t, "", smsService.senderId)
}

func TestSendSMS(t *testing.T) {
	smsService := NewSMSService("testuser", "testapikey", "testsender")
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	tests := []struct {
		name          string
		to            string
		message       string
		mockResponse  string
		mockStatus    int
		expectedError string
	}{
		{
			name:    "successful SMS send",
			to:      "+254740827150",
			message: "Test message",
			mockResponse: `{
				"SMSMessageData": {
					"Message": "Sent to 1/1",
					"Recipients": [{
						"statusCode": 101,
						"number": "+254740827150",
						"status": "Success",
						"cost": "KES 0.80",
						"messageId": "ATXid_123"
					}]
				}
			}`,
			mockStatus:    http.StatusOK,
			expectedError: "",
		},
		{
			name:    "failed SMS send",
			to:      "+254740827150",
			message: "Test message",
			mockResponse: `{
				"SMSMessageData": {
					"Message": "Invalid API Key",
					"Recipients": [{
						"statusCode": 401,
						"number": "+254740827150",
						"status": "Failed",
						"cost": "KES 0.00",
						"messageId": ""
					}]
				}
			}`,
			mockStatus:    http.StatusUnauthorized,
			expectedError: "SMS failed to send: Failed (code: 401)",
		},
		{
			name:          "empty recipients",
			to:            "+254740827150",
			message:       "Test message",
			mockResponse:  `{"SMSMessageData": {"Message": "No recipients", "Recipients": []}}`,
			mockStatus:    http.StatusOK,
			expectedError: "no recipients in response",
		},
		{
			name:          "malformed JSON response",
			to:            "+254740827150",
			message:       "Test message",
			mockResponse:  `{invalid json}`,
			mockStatus:    http.StatusOK,
			expectedError: "failed to decode response",
		},
		{
			name:          "network error",
			to:            "+254740827150",
			message:       "Test message",
			mockResponse:  "",
			mockStatus:    0,
			expectedError: "failed to send request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Reset()

			if tt.mockStatus != 0 {
				httpmock.RegisterResponder("POST", smsService.baseUrl,
					httpmock.NewStringResponder(tt.mockStatus, tt.mockResponse))
			} else {
				httpmock.RegisterResponder("POST", smsService.baseUrl,
					httpmock.NewErrorResponder(fmt.Errorf("network error")))
			}

			err := smsService.SendSMS(tt.to, tt.message)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			if tt.mockStatus != 0 {
				info := httpmock.GetCallCountInfo()
				assert.Equal(t, 1, info["POST "+smsService.baseUrl])
			}
		})
	}
}
