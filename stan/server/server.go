package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
    "os/signal"
    "syscall"
	"bufio"
	"log"
	"os"
	"github.com/gorilla/websocket"
	"github.com/jamiealquiza/tachymeter"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	uuid "github.com/satori/go.uuid"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type NATSMessage struct {
	ID    string `json:id`
	Value string `json:value`
}

func serveWs(sc stan.Conn, w http.ResponseWriter, r *http.Request) {
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


	// Once Ctrl-C is hit, write all of the samples to a file and exit.
	go func() {
		for {
			select {
	                case <-sigchan:
	                    fmt.Printf("Writing times to file")
                		file, err := os.OpenFile("stan-times.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
         
		                if err != nil {
		                        log.Fatalf("failed creating file: %s", err)
		                }
		         
                		datawriter := bufio.NewWriter(file)
         
                		for _, data := range times {
                        	 _, _ = datawriter.WriteString(fmt.Sprintf("%.2f", float64(data)/float64(1000)) + "\n")

						}
         
		                datawriter.Flush()
		                file.Close()
		                os.Exit(0)
	              }
			}
	}()

	msgID := uuid.Must(uuid.NewV4()).String()
	var ts time.Time
	idx := 0
	times := []int64{}

	// Listen to any messages sent on the reply channel, which are sent from the connector.
	sub, err := sc.Subscribe("reply-topic", func(msg *stan.Msg) {
	    since := time.Since(ts)
		t.AddTime(time.Since(ts))
		times = append(times, since.Microseconds())

		var decoded NATSMessage
		err := json.Unmarshal(msg.Data, &decoded)
		if err != nil {
			panic(err)
		}
		if decoded.ID != msgID {
			return
		}

		err = ws.WriteMessage(websocket.TextMessage, msg.Data)
		if err != nil {
			panic(err)
		}

		idx += 1
		if idx%100 == 0 {
			fmt.Println(t.Calc())
		}
	}, stan.StartWithLastReceived())
	defer sub.Close()

	// Receive messages from the websocket and send out the message to the query-topic.
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			break
		}
		var natsMessage NATSMessage
		natsMessage.ID = msgID
		natsMessage.Value = string(data)
		serialized, err := json.Marshal(natsMessage)
		if err != nil {
			panic(err)
		}
		ts = time.Now()

		err = sc.Publish("query-topic", serialized)
		if err != nil {
			panic(err)
		}
	}
}

func main() {
	// REPLACE THIS WITH YOUR NATS ADDRESS.
	natsAddr := "nats://example-nats:4222"
	serverAddr := ":7777"

	nc, err := nats.Connect(natsAddr)
	if err != nil {
		panic(err)
	}

	sc, err := stan.Connect("example-stan", "client-123", stan.NatsConn(nc))
	if err != nil {
		panic(err)
	}
	defer sc.Close()

	// This endpoint can be used to ensure you can connect to the server.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello"))
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(sc, w, r)
	})

	fmt.Printf("Starting server on: %s\n", serverAddr)
	if err := http.ListenAndServe(serverAddr, nil); err != nil {
		panic(err)
	}
}
