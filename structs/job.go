package structs

import "net/http"

const (
	ADD    JobberType = 0
	DELETE JobberType = 1
)

type JobberType uint8

type HTTPRequestDTO struct {
	URL     string         `json:"url"`
	Method  string         `json:"method"`
	Headers http.Header    `json:"headers"`
	Body    map[string]any `json:"body"`
}

type Jobber struct {
	Type            JobberType
	QueueName       string
	Payload         *HTTPRequestDTO
	ResponseChannel chan<- any
}
