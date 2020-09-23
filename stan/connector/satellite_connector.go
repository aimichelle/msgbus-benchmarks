package main

import (
	"os"
	"os/signal"
	"sync"
	"fmt"
    "github.com/nats-io/stan.go"
    "log"
	"github.com/nats-io/nats.go"
)

func WaitForCtrlC() {
	var end_waiter sync.WaitGroup
	end_waiter.Add(1)
	var signal_channel chan os.Signal
	signal_channel = make(chan os.Signal, 1)
	signal.Notify(signal_channel, os.Interrupt)
	go func() {
		<-signal_channel
		end_waiter.Done()
	}()
	end_waiter.Wait()
}

func main() {
	// REPLACE THIS WITH YOUR NATS ADDRESS.
	natsAddr := "nats://example-nats:4222"

	nc, err := nats.Connect()
	if err != nil {
		panic(err)
	}

	sc, err := stan.Connect("example-stan", "client-123", stan.NatsConn(nc))
	if err != nil {
		log.Fatal(err)
	}

	sc.Subscribe("query-topic", func(m *stan.Msg) {
             fmt.Printf("Sending response\n")
             err := sc.Publish("reply-topic", m.Data)
              if err != nil {
                      panic(err)
             }
	     m.Ack()
	
	}, stan.DeliverAllAvailable())

	WaitForCtrlC()
}
