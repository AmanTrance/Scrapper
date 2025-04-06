package croner

import (
	"context"
	"database/sql"

	"github.com/go-co-op/gocron/v2"
	amqp "github.com/rabbitmq/amqp091-go"
)

type CronEngine struct {
	scheduler     gocron.Scheduler
	listener      <-chan any
	rabbitChannel *amqp.Channel
	postres       *sql.DB
}

func NewCronEngine(listener <-chan any, rabbitChannel *amqp.Channel, postgres *sql.DB) (*CronEngine, error) {

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

		case _ = <-engine.listener:

		case <-context.Done():
			_ = engine.scheduler.Shutdown()
			break looper
		}
	}
}
