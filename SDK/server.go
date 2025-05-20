package sdk

import (
	"fmt"
	"github.com/PlakarKorp/plakar/SDK/exchanger"
	"google.golang.org/grpc"
	"log"
	"net"
)

type Server struct {
	exchanger.UnimplementedExchangerServer

	Sdk *PlakarImporterSDK
}

func (s *Server) Scan(stream exchanger.Exchanger_ScanServer) error {
	for {
		_, err := stream.Recv()
		if err != nil {
			log.Println("Scan recv err:", err)
			return err
		}
		response, _ := s.Sdk.scan()
		fmt.Printf("Scan response: %v\n", response)
		responseString := fmt.Sprintf("%v", response)
		err = stream.Send(&exchanger.ScanMessage{Data: responseString})
		if err != nil {
			log.Println("Scan send err:", err)
			return err
		}
	}
}

func (s *Server) NewReader(stream exchanger.Exchanger_NewReaderServer) error {
	for {
		in, err := stream.Recv()
		if err != nil {
			log.Println("Reader recv err:", err)
			return err
		}
		dataString := string(in.Chunk)
		fmt.Printf("\nI'll write %s\n", dataString)
		response, err := s.Sdk.NewReader(dataString)
		responseString := fmt.Sprintf("%v", response)
		if err != nil {
			log.Println("NewReader err:", err)
		}
		err = stream.Send(&exchanger.ReaderMessage{Chunk: responseString})
		if err != nil {
			log.Println("NewReader send err:", err)
			return err
		}
		return nil
	}
}

func (s *Server) Run() {
	fmt.Println("Starting gRPC server...")
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Println("Listening on port 50051...")
	srv := grpc.NewServer()
	exchanger.RegisterExchangerServer(srv, s)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
