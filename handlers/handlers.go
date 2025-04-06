package handlers

import (
	"database/sql"
	"net/http"
	"scraper/structs"
)

type Handler struct {
	postgres    *sql.DB
	cronChannel chan<- *structs.Jobber
}

func NewHandler(postgres *sql.DB, cronChannel chan<- *structs.Jobber) *Handler {

	var handler Handler = Handler{postgres, cronChannel}

	return &handler
}

func (h *Handler) AddScraper() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {}
}

func (h *Handler) UpdateScraper() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {}
}

func (h *Handler) DeleteScraper() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {}
}

func (h *Handler) ListScrapers() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {}
}
