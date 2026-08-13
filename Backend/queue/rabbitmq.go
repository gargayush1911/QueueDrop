package queue

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

var Conn *amqp.Connection
var Channel *amqp.Channel

func InitRabbitMQ() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal("RabbitMQ Connection Failes: ", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("failed to open a channel: ", err)
	}

	Conn = conn
	Channel = ch

	// declare the queue — creates it if it doesn't exist, does nothing if it does
	_, err = ch.QueueDeclare(
		"purchase_queue", // name
		true,             // durable — survives a RabbitMQ restart
		false,            // auto-delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // extra arguments
	)
	if err != nil {
		log.Fatal("failed to declare queue: ", err)
	}
	log.Println("RabbitMQ connected , queue ready")

}
