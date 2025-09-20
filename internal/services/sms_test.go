package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatPhoneNumber(t *testing.T) {
	smsSservice := NewSMSService("test", "test", "test")

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
			result := smsSservice.formatPhoneNumber(tt.input)
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
		to := "+254740827150"
		message := "test message"

		err := mockService.SendSMS(to, message)
		assert.NoError(t, err)

		assert.Len(t, mockService.SentMessages, 1)
		assert.Equal(t, to, mockService.SentMessages[0].Message)
	})

	t.Run("send bulk sms", func(t *testing.T) {
		recipients := []string{"+254740827150", "+254111768132", "+25477010234"}
		message := "bulk test message"

		err := mockService.SendBulkSMS(recipients, message)
		assert.NoError(t, err)

		assert.Len(t, mockService.SentMessages, 4)

		for i, recipient := range recipients {
			sentMessage := mockService.SentMessages[i+1]
			assert.Equal(t, recipient, sentMessage.To)
			assert.Equal(t, message, sentMessage.Message)
		}
	})

	t.Run("clear and send new messages", func(t *testing.T) {
		newMockService := NewMockSMSService()

		to := "+254740827150"
		message := "new test message"

		err := newMockService.SendSMS(to, message)
		assert.NoError(t, err)

		assert.Len(t, newMockService.SentMessages, 1)
		assert.Equal(t, to, newMockService.SentMessages[0].To)
		assert.Equal(t, message, newMockService.SentMessages[0].Message)
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
