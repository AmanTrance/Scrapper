package croner

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
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

	restartRows, err := engine.postres.Query(`
		SELECT queue_id, cron FROM cron_details
	`)

	if err != nil {
		log.Default().Fatal(err.Error())
	}

	for {
		if restartRows.Next() {
			var queueName string
			var cron string

			_ = restartRows.Scan(&queueName, &cron)

			j, err := engine.scheduler.NewJob(gocron.CronJob(cron, false), gocron.NewTask(func() {
				rows, err := engine.postres.Query(`
					SELECT job_payload, exchange_name FROM cron_details WHERE queue_id = $1
				`, &queueName)

				if err != nil {
					log.Default().Println(err.Error())
					return
				}

				var payload string
				var exchangeName string

				if rows.Next() {
					_ = rows.Scan(&payload, &exchangeName)
				} else {
					rows.Close()
					return
				}

				bytesPayload, err := base64.StdEncoding.DecodeString(payload)
				if err != nil {
					log.Default().Println(err.Error())
					rows.Close()
					return
				}

				var request structs.HTTPRequest
				err = json.Unmarshal(bytesPayload, &request)
				if err != nil {
					log.Default().Println(err.Error())
					rows.Close()
					return
				}

				newHTTPRequest, err := http.NewRequest(request.Method, request.URL, bytes.NewBuffer(request.Body))
				if err != nil {
					log.Default().Println(err.Error())
					rows.Close()
					return
				}

				newHTTPRequest.Header = request.Headers

				response, err := http.DefaultClient.Do(newHTTPRequest)
				if err != nil {
					log.Default().Println(err.Error())
					rows.Close()
					return
				}

				responseBytes, err := io.ReadAll(response.Body)
				if err != nil {
					log.Default().Println(err.Error())
					rows.Close()
					return
				}

				err = rabbit.SendMessageToSpecificExchange(engine.rabbitChannel, &rabbit.Message{Exchange: exchangeName, QueueName: queueName,
					ContentType: response.Header.Get("Content-Type"), ActualData: responseBytes})

				if err != nil {
					log.Default().Println(err.Error())
					rows.Close()
					return
				}
			}))

			if err != nil {
				log.Default().Println(err.Error())
				continue
			}

			_, err = engine.postres.Exec(`
				UPDATE cron_details SET job_id = $1 WHERE queue_id = $2
			`, j.ID().String(), &queueName)

			if err != nil {
				log.Default().Println(err.Error())
				continue
			}

		} else {
			break
		}
	}

	restartRows.Close()

looper:
	for {
		select {

		case job := <-engine.listener:
			switch job.Type {

			case structs.ADD:
				jobPayload, err := json.Marshal(job.Payload)
				if err != nil {
					log.Default().Println(err.Error())
					break
				}

				base64Payload := base64.StdEncoding.EncodeToString(jobPayload)

				_, err = engine.postres.Exec(`
					INSERT INTO cron_details (queue_id, cron, job_payload, exchange_name) VALUES (
						$1, $2, $3, $4
					)
				`, &job.QueueName, &job.Cron, &base64Payload, &job.ExchangeName)

				if err != nil {
					log.Default().Println(err.Error())
					break
				}

				j, err := engine.scheduler.NewJob(gocron.CronJob(job.Cron, false), gocron.NewTask(func() {
					rows, err := engine.postres.Query(`
						SELECT job_payload, exchange_name FROM cron_details WHERE queue_id = $1
					`, &job.QueueName)

					if err != nil {
						log.Default().Println(err.Error())
						return
					}

					var payload string
					var exchangeName string

					if rows.Next() {
						_ = rows.Scan(&payload, &exchangeName)
					} else {
						rows.Close()
						return
					}

					bytesPayload, err := base64.StdEncoding.DecodeString(payload)
					if err != nil {
						log.Default().Println(err.Error())
						rows.Close()
						return
					}

					var request structs.HTTPRequest
					err = json.Unmarshal(bytesPayload, &request)
					if err != nil {
						log.Default().Println(err.Error())
						rows.Close()
						return
					}

					newHTTPRequest, err := http.NewRequest(request.Method, request.URL, bytes.NewBuffer(request.Body))
					if err != nil {
						log.Default().Println(err.Error())
						rows.Close()
						return
					}

					newHTTPRequest.Header = request.Headers

					response, err := http.DefaultClient.Do(newHTTPRequest)
					if err != nil {
						log.Default().Println(err.Error())
						rows.Close()
						return
					}

					responseBytes, err := io.ReadAll(response.Body)
					if err != nil {
						log.Default().Println(err.Error())
						rows.Close()
						return
					}

					err = rabbit.SendMessageToSpecificExchange(engine.rabbitChannel, &rabbit.Message{Exchange: exchangeName, QueueName: job.QueueName,
						ContentType: response.Header.Get("Content-Type"), ActualData: responseBytes})

					if err != nil {
						log.Default().Println(err.Error())
						rows.Close()
						return
					}
				}))

				if err != nil {
					log.Default().Println(err.Error())
					break
				}

				_, err = engine.postres.Exec(`
					UPDATE cron_details SET job_id = $1 WHERE queue_id = $2
				`, j.ID().String(), &job.QueueName)

				if err != nil {
					log.Default().Println(err.Error())
					break
				}

			case structs.DELETE:
				rows, err := engine.postres.Query(`
					SELECT job_id FROM cron_details WHERE queue_id = $1
				`, &job.QueueName)

				if err != nil {
					log.Default().Println(err.Error())
					break
				}

				var jobID string

				if rows.Next() {
					_ = rows.Scan(&jobID)
				} else {
					rows.Close()
					break
				}

				jobUUID, err := uuid.Parse(jobID)
				if err != nil {
					log.Default().Println(err.Error())
					break
				}

				err = engine.scheduler.RemoveJob(jobUUID)
				if err != nil {
					log.Default().Println(err.Error())
					break
				}

			case structs.UPDATE:
				rows, err := engine.postres.Query(`
					SELECT job_id FROM cron_details WHERE queue_id = $1
				`, &job.QueueName)

				if err != nil {
					log.Default().Println(err.Error())
					break
				}

				var jobID string

				if rows.Next() {
					_ = rows.Scan(&jobID)
				} else {
					rows.Close()
					break
				}

				jobUUID, err := uuid.Parse(jobID)
				if err != nil {
					log.Default().Println(err.Error())
					break
				}

				jobPayload, err := json.Marshal(job.Payload)
				if err != nil {
					log.Default().Println(err.Error())
					break
				}

				base64Payload := base64.StdEncoding.EncodeToString(jobPayload)

				_, err = engine.postres.Exec(`
					UPDATE cron_details SET cron = $1, cron_payload = $2, updated_at = NOW() WHERE queue_id = $3
				`, &job.Cron, &base64Payload, &job.QueueName)

				if err != nil {
					log.Default().Println(err.Error())
					break
				}

				_, err = engine.scheduler.Update(jobUUID, gocron.CronJob(job.Cron, false), gocron.NewTask(func() {
					rows, err := engine.postres.Query(`
						SELECT job_payload, exchange_name FROM cron_details WHERE id = $1
					`, &job.QueueName)

					if err != nil {
						log.Default().Println(err.Error())
						return
					}

					var payload string
					var exchangeName string

					if rows.Next() {
						_ = rows.Scan(&payload, &exchangeName)
					} else {
						rows.Close()
						return
					}

					bytesPayload, err := base64.StdEncoding.DecodeString(payload)
					if err != nil {
						log.Default().Println(err.Error())
						rows.Close()
						return
					}

					var request structs.HTTPRequest
					err = json.Unmarshal(bytesPayload, &request)
					if err != nil {
						log.Default().Println(err.Error())
						rows.Close()
						return
					}

					newHTTPRequest, err := http.NewRequest(request.Method, request.URL, bytes.NewBuffer(request.Body))
					if err != nil {
						log.Default().Println(err.Error())
						rows.Close()
						return
					}

					newHTTPRequest.Header = request.Headers

					response, err := http.DefaultClient.Do(newHTTPRequest)
					if err != nil {
						log.Default().Println(err.Error())
						rows.Close()
						return
					}

					responseBytes, err := io.ReadAll(response.Body)
					if err != nil {
						log.Default().Println(err.Error())
						rows.Close()
						return
					}

					err = rabbit.SendMessageToSpecificExchange(engine.rabbitChannel, &rabbit.Message{Exchange: exchangeName, QueueName: job.QueueName,
						ContentType: response.Header.Get("Content-Type"), ActualData: responseBytes})
					if err != nil {
						log.Default().Println(err.Error())
						rows.Close()
						return
					}
				}))

				if err != nil {
					log.Default().Println(err.Error())
					break
				}
			}

		case <-context.Done():
			_ = engine.scheduler.Shutdown()
			break looper
		}
	}
}
