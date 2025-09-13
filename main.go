package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/mfutselaar/gocliselect"
)

const keys string = "1234567890abcdefghijklmnoprstuvwxyz"

var (
	skipAfterCommands bool = false
	envKvArray        []string
)

type Config struct {
	Runner        Runner        `json:"runner"`
	AfterCommands []string      `json:"after-commands"`
	Types         []ProjectType `json:"types"`
}

type Runner struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type Parent struct {
	Type        string `json:"type"`
	RunCommands bool   `json:"run-commands"`
}

type ProjectType struct {
	Type     string     `json:"type"`
	Parent   *Parent    `json:"parent"`
	Files    [][]string `json:"files"`
	Commands []string   `json:"commands"`
}

func parseString(s string) string {
	if len(envKvArray) == 0 {
		for _, kvp := range os.Environ() {
			split := strings.Split(kvp, "=")
			if len(split) == 2 {
				envKvArray = append(envKvArray, "$"+split[0], split[1])
			}
		}
	}
	replacer := strings.NewReplacer(envKvArray...)
	return replacer.Replace(s)
}

func (p *Parent) UnmarshalJSON(data []byte) error {
	type Alias Parent
	aux := &struct {
		RunCommands *bool `json:"run-commands"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.RunCommands == nil {
		p.RunCommands = true
	} else {
		p.RunCommands = *aux.RunCommands
	}

	return nil
}

func (pt *ProjectType) UnmarshalJSON(data []byte) error {
	type Alias ProjectType
	aux := &struct {
		Parent   json.RawMessage `json:"parent"`
		Files    json.RawMessage `json:"files"`
		Commands json.RawMessage `json:"commands"`
		*Alias
	}{
		Alias: (*Alias)(pt),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if len(aux.Files) > 0 {
		var files [][]string

		if err := json.Unmarshal(aux.Files, &files); err == nil {
			pt.Files = files
		} else {
			pt.Files = [][]string{}
		}
	}

	if len(aux.Commands) > 0 {
		var commands []string

		if err := json.Unmarshal(aux.Commands, &commands); err == nil {
			pt.Commands = commands
		} else {
			pt.Commands = []string{}
		}
	}

	if len(aux.Parent) > 0 {
		var parentString string
		if err := json.Unmarshal(aux.Parent, &parentString); err == nil {
			pt.Parent = &Parent{
				Type:        parentString,
				RunCommands: true,
			}
		} else {
			var parentObj Parent
			if err := json.Unmarshal(aux.Parent, &parentObj); err != nil {
				return err
			}
			pt.Parent = &parentObj
		}
	}

	return nil
}

func (c *Config) FindProjectType(q string) (ProjectType, error) {
	q = strings.ToLower(q)
	for _, projectType := range c.Types {
		if strings.ToLower(projectType.Type) == q {
			return projectType, nil
		}
	}

	return ProjectType{}, fmt.Errorf("%s is not a configured project type", q)
}

func (c *Config) Picker() string {
	picker := gocliselect.NewMenu("Choose your project type")

	for index, it := range c.Types {
		if index < len(keys) {
			picker.AddItemWithInput(it.Type, strings.ToLower(it.Type), rune(keys[index]))
		} else {
			picker.AddItem(it.Type, strings.ToLower(it.Type))
		}
	}

	picker.AddDivider()
	picker.AddItemWithInput("quit / cancel", "quit", 'q')

	choice, err := picker.Display()
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	return choice.(string)
}

func (c *Config) ExecuteCommands(pt *ProjectType) {
	if pt.Parent != nil && pt.Parent.RunCommands {
		ppt, err := c.FindProjectType(pt.Parent.Type)
		if err != nil {
			fmt.Printf("%v\n", err)
		}

		c.ExecuteCommands(&ppt)
	}

	if len(pt.Commands) == 0 {
		return
	}

	fmt.Printf("\n* Running commands for %s:\n\n", pt.Type)

	for _, command := range pt.Commands {
		cmd := exec.Command(c.Runner.Command, append(c.Runner.Args, parseString(command))...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("%v\n", err)
		}
	}
}

func (c *Config) CopyFiles(pt *ProjectType) {
	if pt.Parent != nil {
		ppt, err := c.FindProjectType(pt.Parent.Type)
		if err != nil {
			fmt.Printf("%v\n", err)
		}

		c.CopyFiles(&ppt)
	}

	if len(pt.Files) == 0 {
		return
	}

	fmt.Printf("\n* Copying files for %s:\n\n", pt.Type)

	for _, file := range pt.Files {
		sourcePath := file[0]
		target := file[1]
		fmode := os.FileMode(0644)

		targetPath := path.Dir(parseString(target))

		var sourceData []byte

		u, err := url.ParseRequestURI(sourcePath)
		if err == nil && (u.Scheme == "http" || u.Scheme == "https") {
			fmt.Printf("  * Downloading %s: ", u.String())

			req, err := http.NewRequest(http.MethodGet, u.String(), nil)
			if err != nil {
				fmt.Printf("%v\n", err)
				continue
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Printf("%v\n", err)
				continue
			}

			if res.StatusCode == 200 {
				resBody, err := io.ReadAll(res.Body)
				res.Body.Close()

				if err == nil {
					sourceData = resBody
				} else {
					fmt.Printf("%v\n", err)
					continue
				}
			}

		} else {
			sourcePath = parseString(sourcePath)
			fmt.Printf("  * Reading %s: ", sourcePath)

			fileInfo, err := os.Stat(sourcePath)
			if err != nil {
				fmt.Printf("%v\n", err)
				continue
			}
			fmode = fileInfo.Mode()
			fileData, err := os.ReadFile(sourcePath)
			if err != nil {
				fmt.Printf("%v\n", err)
				continue
			}
			sourceData = fileData
		}

		fmt.Print("OK\n")

		if targetPath != "" {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				fmt.Printf("%v\n", err)
				continue
			}
		}

		fmt.Printf("  * Writing to %s: ", target)

		if _, err := os.Stat(target); err == nil {
			fmt.Printf("File already exists, skipping\n")
			continue
		}

		if err := os.WriteFile(target, sourceData, fmode); err != nil {
			fmt.Printf("%v\n", err)
			continue
		}

		fmt.Print("OK\n")
	}
}

func (pt *ProjectType) Create(c *Config) {
	cwd, _ := os.Getwd()
	fmt.Printf("\n\nCreating project of type %s in %s\n", pt.Type, cwd)

	c.ExecuteCommands(pt)
	c.CopyFiles(pt)

	fmt.Print("\n* Running after commands:\n\n")
	for _, command := range c.AfterCommands {
		cmd := exec.Command(c.Runner.Command, append(c.Runner.Args, parseString(command))...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("%v\n", err)
		}
	}
}

func main() {
	configPaths := []string{
		"./config.json",
		filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "init-go", "config.json"),
		filepath.Join(os.Getenv("HOME"), ".config", "init-go", "config.json"),
		filepath.Join(os.Getenv("HOME"), ".init-go", "config.json"),
		"/etc/init-go.json",
	}

	fmt.Println("init-go, setup your dev environment in no time!")
	createType := ""
	if len(os.Args) > 1 {
		for index := 1; index < len(os.Args); index++ {
			arg := os.Args[index]
			if strings.ToLower(arg) == "--skip-after-commands" || strings.ToLower(arg) == "-s" {
				skipAfterCommands = true
			} else if strings.ToLower(arg) == "-h" {
				fmt.Println()
				fmt.Printf("Usage: %s <options> <type>\n", os.Args[0])
				fmt.Println()
				fmt.Println("All arguments are optional, if no type is provided a selector will be shown.")
				fmt.Println()
				fmt.Println("                         -h	     This help text")
				fmt.Println(" --skip-after-commands,  -s      Do not run commands in the { \"after-commands\": [] } entry")
				fmt.Println()
				fmt.Println("Types are defined in the config, this file needs to be located in one of the following locations:")
				fmt.Println()
				fmt.Println(" * ./config.json")
				fmt.Println(" * $XDG_CONFIG_HOME/init-go/config.json")
				fmt.Println(" * $HOME/.config/init-go/config.json")
				fmt.Println(" * $HOME/.init-go/config.json")
				fmt.Println(" * /etc/init-go.json")
				fmt.Println()

				return
			} else {
				createType = os.Args[index]
			}
		}
	}

	var configData []byte
	var err error
	for _, configPath := range configPaths {
		configData, err = os.ReadFile(configPath)
		if err == nil {
			break
		}
	}

	if err != nil {
		log.Fatal("Could not find a config file in any location: ", err)
	}

	var config Config
	err = json.Unmarshal(configData, &config)
	if err != nil {
		log.Fatal("Could not parse JSON data: ", err)
	}

	if createType == "" {
		createType = config.Picker()
		if createType == "" || createType == "quit" {
			os.Exit(1)
		}
	}

	selectedType, err := config.FindProjectType(createType)
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	selectedType.Create(&config)
}
