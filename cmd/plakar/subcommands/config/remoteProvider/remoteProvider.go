package remoteProvider

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"github.com/PlakarKorp/plakar/appcontext"
	_ "github.com/rclone/rclone/backend/all"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configfile"
	"os"
	"sort"
	"strconv"
	"strings"
)

var handleProviders = map[string]string{
	"googlephotos": "rclone",
	"googledrive":  "rclone",
	"onedrive":     "rclone",
	"opendrive":    "rclone",
	"s3":           "plakar",
	"ftp":          "plakar",
	"sftp":         "plakar",
	"fs":           "plakar",
}

var rcloneProviderName = map[string]string{
	"googlephotos": "google photos",
	"googledrive":  "drive",
	"onedrive":     "onedrive",
	"opendrive":    "opendrive",
}

var providerConf = map[string][]string{
	"s3":   {"location", "access_key", "secret_access_key", "use_tls"},
	"ftp":  {"location", "username", "password"},
	"sftp": {"location"},
	"fs":   {"location"},
}

var hubResponse = map[string]func(*appcontext.AppContext) error{
	"n": NewRemoteProvider,
	"e": EditRemoteProvider,
	"d": DeleteRemoteProvider,
	"q": func(_ *appcontext.AppContext) error { os.Exit(0); return nil },
	"s": ShowOneConfig,
}

var providerCreate = map[string]func(*appcontext.AppContext, string) error{
	"rclone": newRcloneProvider,
	"plakar": newPlakarProvider,
}

var hubMessage = map[int]string{
	0: "You don't have any current remote\n\n",
	1: "Current remote:\n\nName\n====\n",
}

func RemoteHub(ctx *appcontext.AppContext) error {
	var allRemote = getMapKeys(ctx.Config.Remotes)
	var size = len(allRemote)
	if size > 1 {
		print("Current remotes:\n\nName\n====\n")
	} else {
		print(hubMessage[size])
	}

	var possibleInput = getMapKeys(hubResponse)
	var possibleInputCursor = makeCursor(possibleInput)
	for _, remote := range allRemote {
		fmt.Printf("%s\n", remote)
	}
	for {
		print("\n\ne) Edit existing remote\nn) New remote\nd) Delete remote\ns) Show one\nq) Quit config\n" + possibleInputCursor)
		input := strings.TrimSpace(readInput())
		if input == "" {
			fmt.Println("Remote name cannot be empty. Please try again.")
			print(possibleInputCursor)
			continue
		}
		if contains(possibleInput, input) {
			err := hubResponse[input](ctx)
			if err != nil {
				return err
			}
			break
		}
	}
	return RemoteHub(ctx)
}

func NewRemoteProvider(ctx *appcontext.AppContext) error {
	providerMap := getMapKeys(handleProviders)
	provider, err := listSelection(providerMap)

	if err != nil {
		return fmt.Errorf("failed to select provider: %w", err)
	}
	return providerCreate[handleProviders[provider]](ctx, provider)
}

func newRcloneProvider(ctx *appcontext.AppContext, provider string) error {
	configfile.Install()

	name, err := promptForRemoteName(ctx)
	if err != nil {
		return err
	}

	opts := config.UpdateRemoteOpt{All: true}
	generateName := generateConfigName(provider)

	if _, err := config.CreateRemote(context.Background(), generateName, rcloneProviderName[provider], nil, opts); err != nil {
		return fmt.Errorf("failed to create remote: %w", err)
	}

	ctx.Config.Remotes[name] = map[string]string{
		"location": provider + "://" + generateName + ":",
		"service":  "rclone",
	}
	if err := ctx.Config.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	return nil
}

func newPlakarProvider(ctx *appcontext.AppContext, provider string) error {
	name, err := promptForRemoteName(ctx)
	if err != nil {
		return err
	}
	ctx.Config.Remotes[name] = map[string]string{}
	if err := ctx.Config.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	for i := 0; i < len(providerConf[provider]); i++ {
		fmt.Printf("Enter value for %s: ", providerConf[provider][i])
		value := strings.TrimSpace(readInput())
		if value == "" {
			fmt.Println("Value cannot be empty. Please try again.")
			i--
			continue
		}
		ctx.Config.Remotes[name][providerConf[provider][i]] = value
	}
	if err := ctx.Config.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	return nil
}

func EditRemoteProvider(ctx *appcontext.AppContext) error {
	configName, err := listSelection(getMapKeys(ctx.Config.Remotes))

	if err != nil {
		return fmt.Errorf("failed to select profile: %w", err)
	}
	provider, _, _ := strings.Cut(ctx.Config.Remotes[configName]["location"], "://")
	if contains(getMapKeys(rcloneProviderName), provider) {
		rcloneProfile, err := getRcloneProfileByPlakarName(ctx, configName)
		if err != nil {
			return fmt.Errorf("failed to get rclone profile: %w", err)
		}
		configfile.Install()
		return config.EditRemote(context.Background(), nil, rcloneProfile)
	}
	printRemoteConf(configName, ctx)
	print("Write <key value> to create a new field or modify it if its from an existing one. \"q\" to leave\n")
	print("> ")
	keyInput, ValueInput, _ := strings.Cut(strings.TrimSpace(readInput()), " ")
	for keyInput != "q" {
		for key, _ := range ctx.Config.Remotes[configName] {
			if keyInput == key {
				ctx.Config.Remotes[configName][key] = ValueInput
				err := ctx.Config.Save()
				if err != nil {
					return err
				}
				printRemoteConf(configName, ctx)
			}
		}
		print("> ")
		keyInput, ValueInput, _ = strings.Cut(strings.TrimSpace(readInput()), " ")
	}

	return nil
}

func DeleteRemoteProvider(ctx *appcontext.AppContext) error {
	configName, err := listSelection(getMapKeys(ctx.Config.Remotes))

	if err != nil {
		return fmt.Errorf("failed to select profile: %w", err)
	}
	if ctx.Config.Remotes[configName]["service"] == "rclone" {
		rcloneProfile, err := getRcloneProfileByPlakarName(ctx, configName)
		if err != nil {
			return fmt.Errorf("failed to get rclone profile: %w", err)
		}
		configfile.Install()
		config.DeleteRemote(rcloneProfile)
	}
	delete(ctx.Config.Remotes, configName)
	if err := ctx.Config.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}

func printRemoteConf(configName string, ctx *appcontext.AppContext) {
	print("\n" + configName + " data in (Key: Value)\n")
	for key, value := range ctx.Config.Remotes[configName] {
		fmt.Printf("    %s:\t%s\n", key, value)
	}
	print("\n")
}

func ShowOneConfig(ctx *appcontext.AppContext) error {
	configName, err := listSelection(getMapKeys(ctx.Config.Remotes))
	if err != nil {
		return fmt.Errorf("failed to select profile: %w", err)
	}
	printRemoteConf(configName, ctx)
	return nil
}

func makeCursor(possibleInput []string) string {
	var cursor string
	for i, input := range possibleInput {
		cursor += input
		if i != len(possibleInput)-1 {
			cursor += "/"
		}
	}
	cursor += "> "
	return cursor
}

func getMapKeys[T any](remotes map[string]T) []string {
	keys := make([]string, 0, len(remotes))
	for key := range remotes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func listSelection(list []string) (string, error) {
	if len(list) == 0 {
		return "", fmt.Errorf("no profiles found")
	}
	fmt.Println("Enter the number that corresponds to your choice.")
	for i := 0; i < len(list); i++ {
		fmt.Printf("%d: %s\n", i+1, list[i])
	}
	for {
		fmt.Print("\n> ")
		choice, err := strconv.Atoi(strings.TrimSpace(readInput()))
		if err == nil && choice > 0 && choice <= len(list) {
			return list[choice-1], nil
		}
		fmt.Println("Invalid choice. Please try again.")
	}
}

func getRcloneProfileByPlakarName(ctx *appcontext.AppContext, plakarName string) (string, error) {
	if !ctx.Config.HasRemote(plakarName) {
		return "", fmt.Errorf("remote %s not found", plakarName)
	}
	rcloneName := ctx.Config.Remotes[plakarName]["location"]
	_, profileName, _ := strings.Cut(rcloneName, "://")
	profileName = strings.TrimSuffix(profileName, ":")
	if profileName == "" {
		return "", fmt.Errorf("failed to get profile name for %s", plakarName)
	}
	return profileName, nil
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func nextRandom() string {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", b)
}

func generateConfigName(provider string) string {
	name := fmt.Sprintf("plakar-%s-%d", provider, os.Getpid())
	for {
		if !config.LoadedData().HasSection(name) {
			return name
		}
		if len(name) > 30 {
			name, _, _ = strings.Cut(name, provider)
			name += provider + "-"
		}
		name = fmt.Sprintf("%s%s", name, nextRandom())
	}
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

func readInput() string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	print("\n") // print a new line to gain more visibility
	return strings.TrimSpace(input)
}
