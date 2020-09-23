package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"bufio"
	"log"
	"os"
	"github.com/gorilla/websocket"
	"github.com/jamiealquiza/tachymeter"
	uuid "github.com/satori/go.uuid"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
        "os/signal"
        "syscall"

)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Msg struct {
	ID    string `json:id`
	Value string `json:value`
}

func serveWs(p *kafka.Producer, c *kafka.Consumer, w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Upgrading connection")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Upgraded\n")

	// Start sampler.
	t := tachymeter.New(&tachymeter.Config{Size: 100})

    sigchan := make(chan os.Signal, 1)
    signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	msgID := uuid.Must(uuid.NewV4()).String()

	var ts time.Time
	idx := 0
	times := []int64{}

	// Once Ctrl-C is hit, write all of the samples to a file and exit.
	go func() {
		for {
			select {
	            case <-sigchan:
	                fmt.Printf("Writing times to file")
                	file, err := os.OpenFile("kafka-times.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
         
	                if err != nil {
	                        log.Fatalf("failed creating file: %s", err)
	                }
	         
	                datawriter := bufio.NewWriter(file)
	         
	                for _, data := range times {
	                        _, _ = datawriter.WriteString(fmt.Sprintf("%d", data) + "\n")
	                }
         
	                datawriter.Flush()
	                file.Close()
	                os.Exit(3)
		    }
		}
	}()

	// Listen to any messages sent on the reply channel, which are sent from the connector.
	go func() {
		for {
			msg, err := c.ReadMessage(100 * time.Second)
			if err != nil {
				panic(err)
			}
			
			since := time.Since(ts)
			t.AddTime(since)
			times = append(times, since.Milliseconds())

			var decoded Msg
			err = json.Unmarshal(msg.Value, &decoded)
			if err != nil {
				panic(err)
			}
			if decoded.ID != msgID {
				return
			}

			err = ws.WriteMessage(websocket.TextMessage, msg.Value)
			if err != nil {
				panic(err)
			}

			idx += 1
			if idx%100 == 0 {
				fmt.Println(t.Calc())
			}
		}
	}()

	// Receive messages from the websocket and send out the message to the query-topic.
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			break
		}
		var msg Msg
		msg.ID = msgID
		msg.Value = string(data)
		serialized, err := json.Marshal(natsMessage)
		if err != nil {
			panic(err)
		}

		ts = time.Now()
		topic := "query-topic"
		p.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic,
				Partition: 0},
			Value: []byte(serialized)}, nil)

		// Wait for delivery report
		e := <-p.Events()

		message := e.(*kafka.Message)
		if message.TopicPartition.Error != nil {
			fmt.Printf("failed to deliver message: %v\n",
				message.TopicPartition)
		} else {
			fmt.Printf("delivered to topic %s [%d] at offset %v\n",
				*message.TopicPartition.Topic,
				message.TopicPartition.Partition,
				message.TopicPartition.Offset)
		}

	}
}

func main() {
	serverAddr := ":7777"
	// REPLACE THIS WITH YOUR KAFKA ADDRESS.
	kafkaServer := "<KAFKA_ADDRESS>:9092"

	// Produce a new record to the topic...
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": kafkaServer,
	})

	if err != nil {
		panic(fmt.Sprintf("Failed to create producer: %s", err))
	}

	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  kafkaServer,
		"session.timeout.ms": 6000,
		"group.id":           "my-group",
		"auto.offset.reset":  "earliest"})

	if err != nil {
		panic(err)
	}

	topics := []string{"reply-topic"}
	consumer.SubscribeTopics(topics, nil)

	// This endpoint can be used to ensure you can connect to the server.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello"))
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(producer, consumer, w, r)
	})

	fmt.Printf("Starting server on: %s\n", serverAddr)
	if err := http.ListenAndServe(serverAddr, nil); err != nil {
		panic(err)
	}
}
