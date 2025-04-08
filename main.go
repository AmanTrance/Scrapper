package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"scraper/croner"
	"scraper/handlers"
	rabbit "scraper/rabbitmq"
	"scraper/structs"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

func main() {

	systemContext, cancel := context.WithCancel(context.Background())
	configFile, err := os.ReadFile("./config.json")
	if err != nil {
		log.Default().Fatal(err.Error())
	}

	var config structs.Config
	err = json.Unmarshal(configFile, &config)
	if err != nil {
		log.Default().Fatal(err.Error())
	}

postgresRetry:
	postgresClient, err := sql.Open("postgres", config.PostgresURL)
	if err != nil {
		log.Default().Println(err.Error())
		time.Sleep(time.Minute)
		goto postgresRetry
	}

rabbitRetry:
	rabbitClient, err := rabbit.NewRabbitMQClient(&config)
	if err != nil {
		log.Default().Println(err.Error())
		time.Sleep(time.Minute)
		goto rabbitRetry
	}

	err = rabbit.SetupExchanges(rabbitClient, config.Exchange)
	if err != nil {
		time.Sleep(time.Minute)
		goto rabbitRetry
	}

	var cronChannel chan *structs.Jobber = make(chan *structs.Jobber, 1000)

	cronEngine, err := croner.NewCronEngine(cronChannel, rabbitClient, postgresClient)
	if err != nil {
		log.Default().Fatal(err.Error())
	}

	go func() {
		cronEngine.Run(systemContext)
	}()

	var handler *handlers.Handler = handlers.NewHandler(postgresClient, cronChannel, &config)

	http.HandleFunc("/api/v1/add", handler.AddScraper())
	http.HandleFunc("/api/v1/update", handler.UpdateScraper())
	http.HandleFunc("/api/v1/delete", handler.DeleteScraper())
	http.HandleFunc("/api/v1/list", handler.ListScrapers())

	var server http.Server = http.Server{Addr: ":9000", Handler: http.DefaultServeMux}

	go func() {
		err = server.ListenAndServe()
		log.Default().Println(err.Error())
	}()

	var notify chan os.Signal = make(chan os.Signal, 1)
	signal.Notify(notify, os.Interrupt, os.Kill, syscall.SIGINT, syscall.SIGTERM)
	<-notify

	cancel()

	time.Sleep(time.Minute)
}
