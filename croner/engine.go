package croner

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	rabbit "scraper/rabbitmq"
	"scraper/structs"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type CronEngine struct {
	scheduler     gocron.Scheduler
	listener      <-chan *structs.Jobber
	rabbitChannel *amqp.Channel
	postres       *sql.DB
}

func NewCronEngine(listener <-chan *structs.Jobber, rabbitChannel *amqp.Channel, postgres *sql.DB) (*CronEngine, error) {

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	var engine CronEngine = CronEngine{
		scheduler,
		listener,
		rabbitChannel,
		postgres,
	}

	return &engine, nil
}

func (engine *CronEngine) Run(context context.Context) {

	engine.scheduler.Start()

looper:
	for {
		select {

		case job := <-engine.listener:
			switch job.Type {

			case structs.ADD:
				jobPayload, err := json.Marshal(job.Payload)
				if err != nil {
					break
				}

				base64Payload := base64.StdEncoding.EncodeToString(jobPayload)
				id := uuid.NewString()

				_, err = engine.postres.Exec(`
					INSERT INTO cron_details (id, cron, job_payload, queue_name, exchange_name) VALUES (
						$1, $2, $3, $4
					)
				`, &id, &job.Cron, &base64Payload, &job.QueueName, &job.ExchangeName)

				if err != nil {
					break
				}

				_, err = engine.scheduler.NewJob(gocron.CronJob(job.Cron, false), gocron.NewTask(func() {
					rows, err := engine.postres.Query(`
						SELECT job_payload, queue_name, exchange_name FROM cron_details WHERE id = $1
					`, &id)

					if err != nil {
						return
					}

					defer rows.Close()

					var payload string
					var queueName string
					var exchangeName string

					if rows.Next() {
						_ = rows.Scan(&payload, &queueName, &exchangeName)
					}

					bytesPayload, err := base64.StdEncoding.DecodeString(payload)
					if err != nil {
						return
					}

					var request structs.HTTPRequest
					err = json.Unmarshal(bytesPayload, &request)
					if err != nil {
						return
					}

					newHTTPRequest, err := http.NewRequest(request.Method, request.URL, bytes.NewBuffer(request.Body))
					if err != nil {
						return
					}

					newHTTPRequest.Header = request.Headers

					response, err := http.DefaultClient.Do(newHTTPRequest)
					if err != nil {
						return
					}

					responseBytes, err := io.ReadAll(response.Body)
					if err != nil {
						return
					}

					err = rabbit.SendMessageToSpecificExchange(engine.rabbitChannel, &rabbit.Message{Exchange: exchangeName, QueueName: queueName,
						ContentType: response.Header.Get("Content-Type"), ActualData: responseBytes})
					if err != nil {
						return
					}
				}))

				if err != nil {
					break
				}

			case structs.DELETE:

			}

		case <-context.Done():
			_ = engine.scheduler.Shutdown()
			break looper
		}
	}
}
