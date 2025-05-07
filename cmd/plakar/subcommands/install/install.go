package install

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Install{} }, subcommands.BeforeRepositoryWithStorage, "install")
}

const plakarPackageUrl = "https://api.github.com/repos/PlakarKorp/plugins/contents"
const branchName = "?ref=alban/Install-subcommand-test" //TODO: remove the branch name
var pluginFolder = os.Getenv("HOME") + "/.plakar/plugins"

type Install struct {
	subcommands.SubcommandBase

	args []string
}

func (cmd *Install) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("install", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: plakar %s [www.url.to.package]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       plakar %s [package]\n", flags.Name())
		flags.PrintDefaults()
	}

	flags.Parse(args)
	cmd.args = flags.Args()
	return nil
}

type PackageFolderInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Sha         string `json:"sha"`
	Size        int    `json:"size"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	GitURL      string `json:"git_url"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
}

func (cmd *Install) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if len(cmd.args) == 0 {
		return 1, fmt.Errorf("no package specified")
	}

	packageName, versionName, found := strings.Cut(cmd.args[0], "@")
	if !found {
		versionName = "latest"
	} else {
		if versionName == "" {
			return 1, fmt.Errorf("version name cannot be empty")
		}
		if alreadyInstalled(packageName, versionName) {
			return 0, nil
		}
	}
	response, err := getFolderContent(packageName)
	if err != nil {
		return 1, err
	}
	versionFolder, err := getVersionFolder(response, versionName, packageName)
	if versionFolder == "" && err == nil {
		return 0, nil
	}
	if err != nil {
		return 1, err
	}
	response, err = getFolderContent(versionFolder)
	if err != nil {
		return 1, err
	}
	if len(response) != 1 {
		return 1, fmt.Errorf("an error occurred while getting the download URL")
	}
	return downloadPackage(response[0].DownloadURL, versionFolder, response[0].Name)
}

func alreadyInstalled(packageName string, versionName string) bool {
	installedPath := pluginFolder + "/" + packageName + "/" + versionName
	if _, err := os.Stat(installedPath); os.IsNotExist(err) {
		return false
	}
	fmt.Printf("%s is already installed.\nDo you want to reinstall it?\n(y/n)> ", packageName+"@"+versionName)
	var answer string
	fmt.Scanln(&answer)
	for answer != "y" && answer != "Y" && answer != "n" && answer != "N" {
		fmt.Printf("Please answer y or n: ")
		fmt.Scanln(&answer)
	}
	if answer == "y" || answer == "Y" {
		fmt.Printf("Removing %s at %s\n", packageName+"@"+versionName, installedPath)
		os.RemoveAll(installedPath)
		return false
	}
	return true
}

func getFolderContent(folder string) ([]PackageFolderInfo, error) {
	var url = fmt.Sprintf("%s/%s%s", plakarPackageUrl, folder, branchName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get package info: %s", resp.Status)
	}
	defer resp.Body.Close()
	var response []PackageFolderInfo
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	return response, nil
}

func getVersionFolder(response []PackageFolderInfo, versionName string, packageName string) (string, error) {
	versions := make([]*semver.Version, 0)
	versionMap := make(map[string]string)

	for _, folder := range response {
		v, err := semver.NewVersion(folder.Name)
		if err != nil {
			continue
		}
		versions = append(versions, v)
		versionMap[v.String()] = folder.Path
	}

	if versionName == "latest" {
		if len(versions) == 0 {
			return "", fmt.Errorf("no version found for package")
		}
		sort.Sort(sort.Reverse(semver.Collection(versions)))
		if alreadyInstalled(packageName, versions[0].String()) {
			return "", nil
		}
		return versionMap[versions[0].String()], nil
	}
	if path, ok := versionMap[versionName]; ok {
		return path, nil
	}
	return "", fmt.Errorf("version %s not found for the package")
}

func downloadPackage(url string, path string, fileName string) (int, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 1, fmt.Errorf("failed to create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 1, fmt.Errorf("failed to send request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return 1, fmt.Errorf("failed to download package: %s", resp.Status)
	}
	defer resp.Body.Close()

	fullPath := pluginFolder + "/" + path + "/" + fileName
	err = os.MkdirAll(filepath.Dir(fullPath), os.ModePerm)
	if err != nil {
		return 1, fmt.Errorf("failed to create directory: %v", err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return 1, fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return 1, fmt.Errorf("failed to write file: %v", err)
	}
	return 0, nil
}
