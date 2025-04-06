package structs

import "net/http"

const (
	ADD    JobberType = 0
	DELETE JobberType = 1
)

type JobberType uint8

type HTTPRequest struct {
	URL     string      `json:"url"`
	Method  string      `json:"method"`
	Headers http.Header `json:"headers"`
	Body    []byte      `json:"body"`
}

type Jobber struct {
	Type         JobberType
	Cron         string
	QueueName    string
	ExchangeName string
	Payload      *HTTPRequest
}
