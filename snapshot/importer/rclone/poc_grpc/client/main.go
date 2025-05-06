package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	pb "plakar-rclone-plugin/plakar-rclone-plugin/proto"

	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure()) // utilise TLS en prod
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewMyServiceClient(conn)

	fmt.Println("\nRunning ConfigStream:")
	runConfigStream(client)
}

func runConfig(client pb.MyServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	stream, err := client.Config(ctx, &pb.ConfigRequest{Type: "s3"})
	if err != nil {
		log.Fatalf("Config failed: %v", err)
	}

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error receiving from stream: %v", err)
		}
		fmt.Println(">>", msg.Content)
	}
}

func runConfigStream(client pb.MyServiceClient) {
	stream, err := client.ConfigStream(context.Background())
	if err != nil {
		log.Fatalf("ConfigStream failed: %v", err)
	}

	go func() {
		for {
			msg, err := stream.Recv()
			if strings.HasPrefix(msg.Content, "Remote created successfully. Config: ") {
				config := parseRemoteConfig(msg.Content)
				if config != nil {
					fmt.Println("Parsed config:", config)
				} else {
					fmt.Println("Failed to parse config.")
				}
				return
			}
			if err == io.EOF {
				log.Println("Server finished sending messages.")
				return
			}
			if err != nil {
				log.Fatalf("Receive error: %v", err)
			}
			fmt.Println(">>", msg.Content)
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Your input: ")
		text, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Input error: %v", err)
		}
		text = text[:len(text)-1] // remove newline
		if text == "exit" {
			break
		}
		if err := stream.Send(&pb.CLIMessage{Content: text}); err != nil {
			log.Fatalf("Send error: %v", err)
		}
	}
}

func parseRemoteConfig(msg string) map[string]map[string]string {
	prefix := "Remote created successfully. Config: "
	if !strings.HasPrefix(msg, prefix) {
		return nil
	}

	configStr := strings.TrimPrefix(msg, prefix)

	// Option 1 : Traitement brut si tu veux garder ça comme string
	fmt.Printf("Contenu brut extrait: |%s|\n", configStr)

	var config map[string]map[string]string
	err := json.Unmarshal([]byte(configStr), &config)
	if err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return nil
	}
	return config
}
