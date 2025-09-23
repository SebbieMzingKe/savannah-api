package services

type SMSServiceInterface interface {
	SendSMS(to, message string) error
	SendBulkSMS(recipients []string, message string) error
}