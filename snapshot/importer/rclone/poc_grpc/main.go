package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	_ "github.com/rclone/rclone/backend/all"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configfile"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
	"os"
	pb "plakar-rclone-plugin/plakar-rclone-plugin/proto"
	"strconv"
	"strings"
	"sync"
)

var providerList = []string{
	"googlephotos",
	"googledrive",
	"onedrive",
	"opendrive",
	"iclouddrive",
	"s3",
}

type Server struct {
	pb.UnimplementedMyServiceServer
}

func (s *Server) Importer(ctx context.Context, req *pb.KeyValueMap) (*pb.Response, error) {
	log.Println("Importer received:", req.Data)
	// traitement ici...
	return &pb.Response{Status: "Import successful"}, nil
}

func (s *Server) Exporter(ctx context.Context, req *pb.KeyValueMap) (*pb.Response, error) {
	log.Println("Exporter received:", req.Data)
	// traitement ici...
	return &pb.Response{Status: "Export successful"}, nil
}

func (s *Server) Config(req *pb.ConfigRequest, stream pb.MyService_ConfigServer) error {
	log.Println("Config received type:", req.Type)
	return nil
}

func (s *Server) ConfigStream(stream pb.MyService_ConfigStreamServer) error {
	stdinReader, stdinWriter, _ := os.Pipe()
	stdoutReader, stdoutWriter, _ := os.Pipe()

	originalStdin := os.Stdin
	originalStdout := os.Stdout
	originalStderr := os.Stderr

	os.Stdin = stdinReader
	os.Stdout = stdoutWriter
	os.Stderr = stdoutWriter

	defer func() {
		os.Stdin = originalStdin
		os.Stdout = originalStdout
		os.Stderr = originalStderr
		stdinWriter.Close()
		stdoutWriter.Close()
	}()

	inputChan := make(chan string)

	go func() {
		for {
			clientMsg, err := stream.Recv()
			if err == io.EOF {
				close(inputChan)
				return
			}
			if err != nil {
				log.Println("Error receiving from client:", err)
				close(inputChan)
				return
			}
			inputChan <- clientMsg.Content
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stdoutReader)
		for scanner.Scan() {
			msg := scanner.Text()
			stream.Send(&pb.CLIMessage{Content: msg})
		}
	}()

	name, err := promptForRemoteName(stream, inputChan)
	if err != nil {
		return err
	}

	provider, err := listSelection(stream, inputChan, providerList)
	if err != nil {
		return err
	}

	go func() {
		for input := range inputChan {
			_, err := io.WriteString(stdinWriter, input+"\n")
			if err != nil {
				log.Println("Error writing to stdinWriter:", err)
				return
			}
		}
	}()

	configfile.Install()
	tempFile, err := createTempConf()
	if err != nil {
		return err
	}
	defer DeleteTempConf(tempFile.Name())

	opts := config.UpdateRemoteOpt{
		All:       true,
		NoObscure: false,
		Obscure:   true,
	}
	if _, err := config.CreateRemote(context.Background(), name, provider, nil, opts); err != nil {
		return fmt.Errorf("failed to create remote: %w", err)
	}

	jsonConf, err := json.Marshal(getConfFileMap())
	stream.Send(&pb.CLIMessage{Content: fmt.Sprintf("Remote created successfully. Config: %s", string(jsonConf))})
	return nil
}

func promptForRemoteName(stream pb.MyService_ConfigStreamServer, inputChan <-chan string) (string, error) {
	for {
		stream.Send(&pb.CLIMessage{Content: "Choose your remote name: "})

		input, ok := <-inputChan
		if !ok {
			return "", fmt.Errorf("input channel closed unexpectedly")
		}

		name := strings.TrimSpace(input)
		if name == "" {
			stream.Send(&pb.CLIMessage{Content: "Remote name cannot be empty. Please try again."})
			continue
		}
		return name, nil
	}
}

func listSelection(stream pb.MyService_ConfigStreamServer, inputChan <-chan string, list []string) (string, error) {
	if len(list) == 0 {
		return "", fmt.Errorf("no items to choose from")
	}

	stream.Send(&pb.CLIMessage{Content: "Enter the number that corresponds to your choice."})
	for i, item := range list {
		stream.Send(&pb.CLIMessage{Content: fmt.Sprintf("%d: %s", i+1, item)})
	}

	for {
		stream.Send(&pb.CLIMessage{Content: "\n> "})

		input, ok := <-inputChan
		if !ok {
			return "", fmt.Errorf("input channel closed unexpectedly")
		}

		choice, err := strconv.Atoi(strings.TrimSpace(input))
		if err == nil && choice > 0 && choice <= len(list) {
			return list[choice-1], nil
		}

		stream.Send(&pb.CLIMessage{Content: "Invalid choice. Please try again."})
	}
}

func createTempConf() (*os.File, error) {
	tempFile, err := os.CreateTemp("", "rclone-*.conf")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary config file: %w", err)
	}
	err = config.SetConfigPath(tempFile.Name())
	if err != nil {
		return nil, err
	}
	return tempFile, nil
}

func DeleteTempConf(name string) {
	err := os.Remove(name)
	if err != nil {
		fmt.Printf("Error removing temporary file: %v\n", err)
	}
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterMyServiceServer(grpcServer, &Server{})

	log.Println("gRPC server started on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func getConfFileMap() map[string]map[string]string {
	inputFile := config.GetConfigPath()
	file, err := os.Open(inputFile)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return nil
	}
	defer file.Close()

	data := make(map[string]map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	scanner := bufio.NewScanner(file)

	var currentSection string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = line[1 : len(line)-1]
			mu.Lock()
			data[currentSection] = make(map[string]string)
			mu.Unlock()
		} else if currentSection != "" {
			wg.Add(1)
			go func(line, section string) {
				defer wg.Done()
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					mu.Lock()
					data[section][key] = value
					mu.Unlock()
				}
			}(line, currentSection)
		}
	}

	wg.Wait()

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return nil
	}
	return data
}
