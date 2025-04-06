package rabbit

import (
	"scraper/structs"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	DIRECT = "direct"
)

type Message struct {
	Exchange    string
	QueueName   string
	ContentType string
	ActualData  []byte
}

func NewRabbitMQClient(config *structs.Config) (*amqp.Channel, error) {

	connection, err := amqp.Dial(config.RabbitURL)
	if err != nil {
		return nil, err
	}

	channel, err := connection.Channel()
	if err != nil {
		return nil, err
	}

	return channel, nil
}

func SetupExchanges(r *amqp.Channel, exchanges []string) error {

	for _, e := range exchanges {

		err := r.ExchangeDeclare(string(e), DIRECT, true, false, false, false, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func SendMessageToSpecificExchange(r *amqp.Channel, message *Message) error {

	err := r.Publish(message.Exchange, message.QueueName, false, false, amqp.Publishing{ContentType: message.ContentType, Timestamp: time.Now(),
		Expiration: amqp.NeverExpire, Body: message.ActualData})

	if err != nil {
		return err
	}

	return nil
}
