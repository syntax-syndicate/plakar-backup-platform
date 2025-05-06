package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/PlakarKorp/plakar/appcontext"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"strings"
	"sync"

	_ "github.com/rclone/rclone/backend/all"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configfile"
	"google.golang.org/grpc/status"
	"log"
	"net"
	"os"
	"os/exec"

	"gorpc/generated"
)

type providerInfo struct {
	initFunc   func(*appcontext.AppContext, string) error
	rcloneName string
}

var providerList = map[string]providerInfo{}

type GreeterServer struct {
	generated.UnimplementedGreeterServer
}

func newRcloneProvider(ctx *appcontext.AppContext, provider string) error {
	configfile.Install()
	tempFile, err := createTempConf()
	if err != nil {
		return err
	}
	defer DeleteTempConf(tempFile.Name())

	name, err := promptForRemoteName(ctx)
	if err != nil {
		return err
	}

	opts := config.UpdateRemoteOpt{All: true}

	if _, err := config.CreateRemote(context.Background(), name, providerList[provider].rcloneName, nil, opts); err != nil {
		return fmt.Errorf("failed to create remote: %w", err)
	}

	configMap := getConfFileMap()
	ctx.Config.Remotes[name] = map[string]string{}
	for key, value := range configMap[name] {
		ctx.Config.Remotes[name][key] = value
	}
	ctx.Config.Remotes[name]["location"] = provider + "://" + name + ":"

	if err := ctx.Config.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	return nil
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

func readInput() string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	print("\n") // print a new line to gain more visibility
	return strings.TrimSpace(input)
}

func promptForRemoteName(ctx *appcontext.AppContext) (string, error) {
	for {
		fmt.Print("Choose your remote name: ")
		name := strings.TrimSpace(readInput())
		if name == "" {
			fmt.Println("Remote name cannot be empty. Please try again.")
			continue
		}
		if ctx.Config.HasRemote(name) {
			fmt.Printf("Remote %s already exists. Please choose a different name.\n", name)
			continue
		}
		return name, nil
	}
}

func WriteRcloneConfigFile(name string, remoteMap map[string]string) (*os.File, error) {
	file, err := createTempConf()
	_, err = fmt.Fprintf(file, "[%s]\n", name)
	if err != nil {
		return nil, err
	}
	for k, v := range remoteMap {
		_, err = fmt.Fprintf(file, "%s = %s\n", k, v)
	}
	return file, nil
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

func (s *GreeterServer) SayHello(ctx context.Context, req *generated.HelloRequest) (*generated.HelloResponse, error) {
	return &generated.HelloResponse{Message: "Hello, " + req.GetName()}, nil
}

func (s *GreeterServer) DownloadPhoto(ctx context.Context, req *generated.CredentialsApple) (*generated.DownloadPhotoResponse, error) {
	// Spécifie le dossier où les photos seront téléchargées
	directory := "/tmp/plakar_icloud"

	// Crée le dossier si il n'existe pas
	err := os.MkdirAll(directory, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la création du dossier: %v", err)
	}

	// Exécute la commande pour télécharger les photos avec les identifiants Apple
	//icloudpd --username "peralban44@gmail.com" --password "" --directory /tmp/plakar_icloud
	cmd := exec.Command("icloudpd", "--username", req.Email, "--password", req.Password, "--directory", "/tmp/plakar_icloud")

	cmd.Env = os.Environ() // 👈 hérite de l'environnement actuel (comme ton terminal)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "erreur icloudpd: %v\nstderr: %s\nstdout: %s", err, stderr.String(), out.String())
	}

	// Capture les erreurs et la sortie standard de la commande
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("erreur lors de l'exécution de icloudpd: %v", err)
	}

	// Retourne le message de succès
	log.Printf("Sortie de la commande icloudpd: %s", output)
	return &generated.DownloadPhotoResponse{Message: "Photos téléchargées avec succès!"}, nil
}

func main() {
	providerList["googlephotos"] = providerInfo{
		initFunc:   newRcloneProvider,
		rcloneName: "google photos",
	}
	providerList["googledrive"] = providerInfo{
		initFunc:   newRcloneProvider,
		rcloneName: "drive",
	}
	providerList["onedrive"] = providerInfo{
		initFunc:   newRcloneProvider,
		rcloneName: "onedrive",
	}
	providerList["opendrive"] = providerInfo{
		initFunc:   newRcloneProvider,
		rcloneName: "opendrive",
	}
	providerList["iclouddrive"] = providerInfo{
		initFunc:   newRcloneProvider,
		rcloneName: "iclouddrive",
	}
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	generated.RegisterGreeterServer(s, &GreeterServer{})

	reflection.Register(s)
	// Enregistre le service gRPC
	// generated.RegisterGreeterServer(s, &GreeterServer{})

	fmt.Println("Serveur gRPC démarré sur le port :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
