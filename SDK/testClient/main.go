package main

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/PlakarKorp/plakar/SDK/exchanger"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("client dial error: %v", err)
	}
	defer conn.Close()
	client := exchanger.NewExchangerClient(conn)

	for {
		scanStream, err := client.Scan(context.Background())
		if err != nil {
			log.Printf("error opening Scan stream: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		err = scanStream.Send(&exchanger.ScanMessage{Data: "client: scan please"})
		if err != nil {
			log.Printf("error sending scan request: %v", err)
			continue
		}

		msg, err := scanStream.Recv()
		if err == io.EOF {
			log.Println("Scan stream closed by server.")
			continue
		}
		if err != nil {
			log.Printf("error reading from Scan stream: %v", err)
			continue
		}

		log.Printf("Scan response: %s", msg.Data)

		go func(content string) {
			log.Println("Triggering NewReader stream with content:", content)

			newReaderStream, err := client.NewReader(context.Background())
			if err != nil {
				log.Printf("error opening NewReader stream: %v", err)
				return
			}

			err = newReaderStream.Send(&exchanger.ReaderMessage{Chunk: "reader input from scan: " + content})
			if err != nil {
				log.Printf("error sending reader chunk: %v", err)
				return
			}

			resp, err := newReaderStream.Recv()
			if err == io.EOF {
				log.Println("NewReader stream closed.")
				return
			}
			if err != nil {
				log.Printf("error reading from NewReader stream: %v", err)
				return
			}
			log.Printf("NewReader response: %s", resp.Chunk)

		}(msg.Data)

		time.Sleep(1 * time.Second)
	}
}
