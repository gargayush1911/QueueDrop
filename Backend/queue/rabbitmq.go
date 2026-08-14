package queue

import (
	"fmt"
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

var Conn *amqp.Connection
var Channel *amqp.Channel

const PurchaseQueue = "purchase_queue"

func InitRabbitMQ() error {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@localhost:5672/"
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatal("RabbitMQ Connection Failes: ", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open a channel: %w", err)
	}

	// declare the queue — creates it if it doesn't exist, does nothing if it does
	_, err = ch.QueueDeclare(
		PurchaseQueue, // name
		true,          // durable — survives a RabbitMQ restart
		false,         // auto-delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // extra arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to declare queue: %w", err)
	}
	Conn = conn
	Channel = ch

	return nil

}

func CloseRabbitMQ() {
	if Channel != nil {
		Channel.Close()
	}
	if Conn != nil {
		Conn.Close()
	}

}
