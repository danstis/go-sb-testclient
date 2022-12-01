package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/BurntSushi/toml"
	"github.com/danstis/go-sb-testclient/internal/version"
)

var (
	configPath = path.Join(".", "config.txt")
	rm         = azservicebus.ReceiveModePeekLock
)

type settings struct {
	PrimaryServiceBus   serviceBusSettings `toml:"primaryServiceBus`
	SecondaryServiceBus serviceBusSettings `toml:"secondaryServiceBus`
	CompleteMessages    bool               `toml:"complete_messages"`
	CheckInterval       time.Duration      `toml:"check_interval"`
}

type serviceBusSettings struct {
	ConnectionString string `toml:"connection_string"`
	Topic            string `toml:"topic"`
	Subscription     string `toml:"subscription"`
}

// getConfig returns the application configuration from the config TOML file.
func getConfig() (settings, error) {
	var s settings
	_, err := toml.DecodeFile(configPath, &s)
	if os.IsNotExist(err) {
		s := settings{
			PrimaryServiceBus: serviceBusSettings{
				ConnectionString: "Endpoint=...",
				Topic:            "topicName",
				Subscription:     "subscriptionName",
			},
			SecondaryServiceBus: serviceBusSettings{
				ConnectionString: "Endpoint=...",
				Topic:            "topicName",
				Subscription:     "subscriptionName",
			},
			CompleteMessages: false,
			CheckInterval:    5 * time.Second,
		}
		_ = updateConfig(s)
		return settings{}, fmt.Errorf("config file %q not found, please populate the newly created file", configPath)
	}
	if err != nil {
		return settings{}, err
	}
	return s, nil
}

// updateConfig sets the configuration settings in the config TOML file.
func updateConfig(s settings) error {
	f, err := os.OpenFile(configPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()
	return toml.NewEncoder(w).Encode(s)
}

// Main entry point for the app.
func main() {
	log.Printf("Started Azure SB test client, version %q", version.Version)

	cfg, err := getConfig()
	if err != nil {
		log.Fatalf("failed to load config file: %v", err)
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
	go cfg.PrimaryServiceBus.processMessages(ctx, cfg.CheckInterval, "primary")
	go cfg.SecondaryServiceBus.processMessages(ctx, cfg.CheckInterval, "secondary")

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
