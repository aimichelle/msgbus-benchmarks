package main

import (
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	"fmt"

	"os"
	"os/signal"
	"syscall"
)

type Msg struct {
	ID    string `json:id`
	Value string `json:value`
}

func main() {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// REPLACE THIS WITH YOUR KAFKA ADDRESS.
	kafkaServer := "<KAFKA_ADDRESS>:9092"

	// Set up Kafka.
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": kafkaServer,
	})

	if err != nil {
		panic(fmt.Sprintf("Failed to create producer: %s", err))
	}

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  kafkaServer,
		"session.timeout.ms": 6000,
		"group.id":           "my-group-3",
		"auto.offset.reset":  "earliest"})

	if err != nil {
		panic(fmt.Sprintf("failed to create consumer: %s", err))
	}

	run := true

	c.SubscribeTopics([]string{"query-topic"}, nil)

	for run == true {
		select {
		case sig := <-sigchan:
			fmt.Printf("Caught signal %v: terminating\n", sig)
			run = false
		default:
			ev := c.Poll(100)
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				// Received a message on the query-topic. Send a response back.
				topic := "reply-topic"
				producer.Produce(&kafka.Message{
					TopicPartition: kafka.TopicPartition{Topic: &topic,
						Partition: 0},
					Value: e.Value}, nil)
				e2 := <-producer.Events()

				message := e2.(*kafka.Message)
				if message.TopicPartition.Error != nil {
					fmt.Printf("failed to deliver message: %v\n",
						message.TopicPartition)
				} else {
					fmt.Printf("delivered to topic %s [%d] at offset %v\n",
						*message.TopicPartition.Topic,
						message.TopicPartition.Partition,
						message.TopicPartition.Offset)
				}
			case kafka.Error:
				fmt.Fprintf(os.Stderr, "%% Error: %v: %v\n", e.Code(), e)
				if e.Code() == kafka.ErrAllBrokersDown {
					run = false
				}
			default:
				fmt.Printf("Ignored %v\n", e)
			}
		}
	}

	fmt.Printf("Closing consumer\n")
	c.Close()
}

