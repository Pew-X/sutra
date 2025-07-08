package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gopkg.in/yaml.v3"

	"github.com/Pew-X/sutra/internal/agent"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "configs/agent.example.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create and start agent
	synapseAgent, err := agent.NewAgent(*config)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	if err := synapseAgent.Start(); err != nil {
		log.Fatalf("Failed to start agent: %v", err)
	}

	// Wait for shutdown signal
	waitForShutdown(synapseAgent)
}

// loadConfig reads and parses the YAML configuration file.
func loadConfig(path string) (*agent.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config agent.Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// waitForShutdown blocks until a shutdown signal is received.
func waitForShutdown(synapseAgent *agent.Agent) {
	// Create channel to receive OS signals
	sigChan := make(chan os.Signal, 1)

	// Register channel to receive specific signals
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Block until signal is received
	sig := <-sigChan
	log.Printf("Received signal: %v", sig)

	// Graceful shutdown
	if err := synapseAgent.Shutdown(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}
