package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	v1 "github.com/Pew-X/sutra/api/v1"
)

var (
	agentAddr string
	timeout   time.Duration
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "sutractl",
		Short: "Sutra Control - CLI for interacting with Sutra agents",
		Long: `sutractl is a command line interface for querying and managing
Sutra knowledge mesh agents.`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&agentAddr, "agent", "localhost:9090", "Address of Sutra agent")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 10*time.Second, "Request timeout")

	// subcommands
	rootCmd.AddCommand(queryCmd())
	rootCmd.AddCommand(ingestCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(healthCmd())
	rootCmd.AddCommand(metricsCmd())
	rootCmd.AddCommand(peersCmd())

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// queryCmd creates the query subcommand
func queryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query <subject> [predicate]",
		Short: "Query knowledge about a subject",
		Long:  "Query the mesh for all knowledge about a subject, optionally filtered by predicate",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			subject := args[0]
			var predicate *string
			if len(args) > 1 {
				predicate = &args[1]
			}

			return queryKnowledge(subject, predicate)
		},
	}

	return cmd
}

// ingestCmd creates the ingest subcommand
func ingestCmd() *cobra.Command {
	var (
		source     string
		confidence float64
	)

	cmd := &cobra.Command{
		Use:   "ingest <subject> <predicate> <object>",
		Short: "Ingest a knowledge packet",
		Long:  "Send a knowledge packet to the mesh",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			subject := args[0]
			predicate := args[1]
			object := args[2]

			return ingestKnowledge(subject, predicate, object, source, float32(confidence))
		},
	}

	cmd.Flags().StringVar(&source, "source", "synctl", "Source identifier for this knowledge")
	cmd.Flags().Float64Var(&confidence, "confidence", 1.0, "Confidence level (0.0-1.0)")

	return cmd
}

// statusCmd creates the status subcommand
func statusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show agent status",
		Long:  "Display the health and status of the connected agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showStatus()
		},
	}

	return cmd
}

// healthCmd creates the health subcommand
func healthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check agent health",
		Long:  "Get detailed health information from the agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			return checkHealth()
		},
	}

	return cmd
}

// metricsCmd creates the metrics subcommand
func metricsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Show agent metrics",
		Long:  "Display performance metrics and statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showMetrics()
		},
	}

	return cmd
}

// peersCmd creates the peers subcommand
func peersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "peers",
		Short: "List mesh peers",
		Long:  "Show information about other agents in the mesh",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listPeers()
		},
	}

	return cmd
}

// Connect to the agent
func connectToAgent() (v1.SynapseServiceClient, *grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, agentAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to agent at %s: %w", agentAddr, err)
	}

	client := v1.NewSynapseServiceClient(conn)
	return client, conn, nil
}

// queryKnowledge queries the mesh for knowledge
func queryKnowledge(subject string, predicate *string) error {
	client, conn, err := connectToAgent()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req := &v1.QueryRequest{
		Subject:   subject,
		Predicate: predicate,
	}

	stream, err := client.Query(ctx, req)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	fmt.Printf("Knowledge about '%s':\n", subject)
	if predicate != nil {
		fmt.Printf("  (filtered by predicate: %s)\n", *predicate)
	}
	fmt.Println()

	count := 0
	for {
		kpak, err := stream.Recv()
		if err != nil {
			break // End of stream
		}

		count++
		fmt.Printf("  %s %s %s\n", kpak.Subject, kpak.Predicate, kpak.Object)
		fmt.Printf("    Source: %s, Confidence: %.2f, ID: %s\n", kpak.Source, kpak.Confidence, kpak.Id)
		fmt.Println()
	}

	if count == 0 {
		fmt.Println("  No knowledge found.")
	} else {
		fmt.Printf("Found %d knowledge packet(s).\n", count)
	}

	return nil
}

// ingestKnowledge sends a knowledge packet to the mesh
func ingestKnowledge(subject, predicate, object, source string, confidence float32) error {
	client, conn, err := connectToAgent()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	stream, err := client.Ingest(ctx)
	if err != nil {
		return fmt.Errorf("failed to start ingest stream: %w", err)
	}

	kpak := &v1.Kpak{
		Subject:    subject,
		Predicate:  predicate,
		Object:     object,
		Source:     source,
		Confidence: confidence,
		Timestamp:  time.Now().Unix(),
	}

	if err := stream.Send(kpak); err != nil {
		return fmt.Errorf("failed to send k-pak: %w", err)
	}

	response, err := stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("failed to receive response: %w", err)
	}

	if response.Accepted > 0 {
		fmt.Printf("✓ Knowledge packet accepted\n")
		fmt.Printf("  %s %s %s\n", subject, predicate, object)
		fmt.Printf("  Source: %s, Confidence: %.2f\n", source, confidence)
	} else {
		fmt.Printf("✗ Knowledge packet rejected\n")
		if len(response.Errors) > 0 {
			fmt.Printf("  Errors: %v\n", response.Errors)
		}
	}

	return nil
}

// showStatus displays agent status information
func showStatus() error {
	// For now, just test connectivity
	// TODO: Implement proper health check when we have the Health RPC

	fmt.Printf("Connecting to agent at %s...\n", agentAddr)

	_, conn, err := connectToAgent()
	if err != nil {
		return err
	}
	defer conn.Close()

	fmt.Printf("✓ Connected successfully\n")
	fmt.Printf("  Agent: %s\n", agentAddr)
	fmt.Printf("  Status: Online\n")

	// TODO: Add more detailed status infos

	return nil
}

// checkHealth gets detailed health information
func checkHealth() error {
	client, conn, err := connectToAgent()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req := &v1.HealthRequest{}
	resp, err := client.Health(ctx, req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	fmt.Printf("Agent Health Status:\n")
	fmt.Printf("  Status: %s\n", resp.Status)
	fmt.Printf("  Knowledge packets: %d\n", resp.KpakCount)
	fmt.Printf("  Uptime: %d seconds\n", resp.UptimeSeconds)

	return nil
}

// showMetrics displays performance metrics
func showMetrics() error {
	client, conn, err := connectToAgent()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req := &v1.MetricsRequest{}
	resp, err := client.GetMetrics(ctx, req)
	if err != nil {
		return fmt.Errorf("metrics request failed: %w", err)
	}

	fmt.Printf("Agent Performance Metrics:\n")
	fmt.Printf("  Version: %s\n", resp.Version)
	fmt.Printf("  Uptime: %d seconds\n", resp.UptimeSeconds)
	fmt.Printf("\nKnowledge Base:\n")
	fmt.Printf("  Total k-paks: %d\n", resp.TotalKpaks)
	fmt.Printf("  Total subjects: %d\n", resp.TotalSubjects)
	fmt.Printf("\nPerformance:\n")
	fmt.Printf("  Ingest rate: %d k-paks/min\n", resp.IngestRatePerMin)
	fmt.Printf("  Query rate: %d queries/min\n", resp.QueryRatePerMin)
	fmt.Printf("\nSystem Resources:\n")
	fmt.Printf("  Memory usage: %.2f MB\n", float64(resp.MemoryUsageBytes)/(1024*1024))
	fmt.Printf("  CPU usage: %.1f%%\n", resp.CpuUsagePercent)
	fmt.Printf("\nActive Sources:\n")
	for _, source := range resp.ActiveSources {
		fmt.Printf("  - %s\n", source)
	}

	return nil
}

// listPeers shows mesh peer information
func listPeers() error {
	client, conn, err := connectToAgent()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req := &v1.PeersRequest{}
	resp, err := client.GetPeers(ctx, req)
	if err != nil {
		return fmt.Errorf("peers request failed: %w", err)
	}

	fmt.Printf("Mesh Peers (%d total):\n", len(resp.Peers))
	if len(resp.Peers) == 0 {
		fmt.Printf("  No peers found (running solo)\n")
		return nil
	}

	for _, peer := range resp.Peers {
		stateStr := "unknown"
		switch peer.State {
		case 0:
			stateStr = "alive"
		case 1:
			stateStr = "suspect"
		case 2:
			stateStr = "dead"
		}

		fmt.Printf("  %s\n", peer.Name)
		fmt.Printf("    Address: %s\n", peer.Address)
		fmt.Printf("    State: %s\n", stateStr)
		fmt.Printf("    Last seen: %d\n", peer.LastSeen)
		fmt.Println()
	}

	return nil
}
