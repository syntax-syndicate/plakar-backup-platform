package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"gorpc/generated"
	"log"
)

func main() {
	// Connexion au serveur gRPC (assurez-vous que le serveur est en marche sur :50051)
	conn, err := grpc.Dial(":50051", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Créer un client Greeter
	client := generated.NewGreeterClient(conn)

	// Appel de la méthode SayHello
	name := "peralban44@gmail.com"
	password := "Incroyable4002"

	req := &generated.CredentialsApple{
		Email:    name,
		Password: password,
	}

	// Appel de la méthode SayHello
	resp, err := client.DownloadPhoto(context.Background(), req)
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}

	fmt.Printf("Message from server: %s\n", resp.GetMessage())
}
