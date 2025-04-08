package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"scraper/structs"

	"github.com/google/uuid"
)

type Handler struct {
	postgres    *sql.DB
	cronChannel chan<- *structs.Jobber
	config      *structs.Config
}

func NewHandler(postgres *sql.DB, cronChannel chan<- *structs.Jobber, config *structs.Config) *Handler {

	var handler Handler = Handler{postgres, cronChannel, config}

	return &handler
}

func (h *Handler) AddScraper() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		defer r.Body.Close()

		var body structs.CreateJobDTO

		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			log.Default().Println(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var queueName string = uuid.NewString()

		h.cronChannel <- &structs.Jobber{Type: structs.ADD, Cron: body.CronExpression, QueueName: queueName,
			ExchangeName: h.config.Exchange, Payload: &body.HTTPRequest}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fmt.Sprintf("{\"queue_name\":\"%s\"}", queueName)))
	}
}

func (h *Handler) UpdateScraper() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		defer r.Body.Close()

		var body structs.UpdateJobDTO

		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			log.Default().Println(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		h.cronChannel <- &structs.Jobber{Type: structs.UPDATE, Cron: body.CronExpression, QueueName: body.QueueName,
			ExchangeName: h.config.Exchange, Payload: &body.HTTPRequest}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}
}

func (h *Handler) DeleteScraper() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		defer r.Body.Close()

		var queueName string = r.URL.Query().Get("queueID")

		h.cronChannel <- &structs.Jobber{Type: structs.DELETE, QueueName: queueName}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}
}

func (h *Handler) ListScrapers() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		defer r.Body.Close()

		rows, err := h.postgres.Query(`
			SELECT queue_id, cron, job_payload FROM cron_details
		`)

		if err != nil {
			log.Default().Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		defer rows.Close()

		var result []map[string]any

		for {
			if rows.Next() {
				var queueName string
				var cron string
				var payload string

				_ = rows.Scan(&queueName, &cron, &payload)

				var temp map[string]any = make(map[string]any)
				temp["queue_name"] = queueName
				temp["cron"] = cron
				temp["payload"] = payload

				result = append(result, temp)
			} else {
				break
			}
		}

		responseBytes, err := json.Marshal(&result)
		if err != nil {
			log.Default().Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	}
}
