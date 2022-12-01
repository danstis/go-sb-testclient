package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/danstis/go-sb-testclient/internal/version"
)

var (
	configPath = path.Join(".", "config.json")
	rm         = azservicebus.ReceiveModePeekLock
)

type settings struct {
	PrimaryServiceBus   serviceBusSettings `json:"primaryServiceBus"`
	SecondaryServiceBus serviceBusSettings `json:"secondaryServiceBus"`
	CompleteMessages    bool               `json:"completeMessages"`
	CheckInterval       string             `json:"checkInterval"`
}

type serviceBusSettings struct {
	ConnectionString string `json:"connectionString"`
	Topic            string `json:"topic"`
	Subscription     string `json:"subscription"`
}

// Main entry point for the app.
func main() {
	log.Printf("Started Azure SB test client, version %q", version.Version)

	// Read the config file.
	fc, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("failed to open config file, please ensure the config file '%s' exists: %v", configPath, err)
	}
	var cfg settings
	err = json.Unmarshal(fc, &cfg)
	if err != nil {
		log.Fatalf("failed to read configuration from file '%s': %v", configPath, err)
	}

	//  Convert the checkInterval to a time.Duration
	checkInt, err := time.ParseDuration(cfg.CheckInterval)
	if err != nil {
		log.Fatalf("couldn't convert '%v' to a duration: %v", cfg.CheckInterval, err)
	}

	// Override the ReceiveMode based on the configuration.
	if cfg.CompleteMessages {
		rm = azservicebus.ReceiveModeReceiveAndDelete
	}

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the go routines for the primary and secondary connections.
	go cfg.PrimaryServiceBus.processMessages(ctx, checkInt, "PRI")
	go cfg.SecondaryServiceBus.processMessages(ctx, checkInt, "SEC")

	<-termChan
}

// processMessages will connect to a service bus and receive any messages.
func (s serviceBusSettings) processMessages(ctx context.Context, ci time.Duration, name string) {
	// See here for instructions on how to get a Service Bus connection string:
	// https://docs.microsoft.com/azure/service-bus-messaging/service-bus-quickstart-portal#get-the-connection-string
	client, err := azservicebus.NewClientFromConnectionString(s.ConnectionString, nil)
	if err != nil {
		log.Printf("[%s] failed to connect to service bus: %v", name, err)
	} else {
		rec, err := client.NewReceiverForSubscription(s.Topic, s.Subscription, &azservicebus.ReceiverOptions{ReceiveMode: rm})
		if err != nil {
			log.Printf("[%s] failed to open service bus subscription: %v", name, err)
			<-ctx.Done()
		}

		for {
			msgs, err := rec.ReceiveMessages(context.Background(), 100, nil)
			if err != nil {
				log.Printf("[%s] failed to get messages: %v", name, err)
				<-ctx.Done()
			}
			for _, m := range msgs {
				log.Printf("[%s] QueuedTime: %v - Body: %s", name, m.EnqueuedTime, m.Body)
			}
			time.Sleep(ci)
		}
	}
}
