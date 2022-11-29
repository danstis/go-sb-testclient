package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/BurntSushi/toml"
	"github.com/danstis/go-sb-testclient/internal/version"
)

var configPath = path.Join(".", "config.txt")

type settings struct {
	ConnectionString string        `toml:"connection_string"`
	Topic            string        `toml:"topic"`
	Subscription     string        `toml:"subscription"`
	CompleteMessages bool          `toml:"complete_messages"`
	CheckInterval    time.Duration `toml:"check_interval"`
}

// getConfig returns the application configuration from the config TOML file.
func getConfig() (settings, error) {
	var s settings
	_, err := toml.DecodeFile(configPath, &s)
	if os.IsNotExist(err) {
		s := settings{
			ConnectionString: "Endpoint=...",
			Topic:            "topicName",
			Subscription:     "subscriptionName",
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

	// Set the ReceiveMode based on the configuration.
	rm := azservicebus.ReceiveModePeekLock
	if cfg.CompleteMessages {
		rm = azservicebus.ReceiveModeReceiveAndDelete
	}

	// See here for instructions on how to get a Service Bus connection string:
	// https://docs.microsoft.com/azure/service-bus-messaging/service-bus-quickstart-portal#get-the-connection-string
	client, err := azservicebus.NewClientFromConnectionString(cfg.ConnectionString, nil)
	if err != nil {
		log.Fatalf("failed to connect to service bus: %v", err)
	}

	rec, err := client.NewReceiverForSubscription(cfg.Topic, cfg.Subscription, &azservicebus.ReceiverOptions{ReceiveMode: rm})
	if err != nil {
		log.Fatalf("failed to open service bus subscription: %v", err)
	}

	for {
		msgs, err := rec.ReceiveMessages(context.Background(), 100, nil)
		if err != nil {
			log.Fatalf("failed to get messages: %v", err)
		}
		for _, m := range msgs {
			log.Printf("QueuedTime: %v - Body: %s", m.EnqueuedTime, m.Body)
		}
		time.Sleep(cfg.CheckInterval)
	}
}
