package ansible

import (
	"fmt"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Hosts string         `yaml:"hosts"`
	Vars  map[string]any `yaml:"vars"`
}

// Define the YAML-like structure as Go structs
type Command struct {
	Description string            `yaml:"description"`
	Options     map[string]Option `yaml:"options"`
}

// Parse command-line arguments
func parseArgs(cmdMap map[string]any, args []string) (string, map[string]string, error) {
	if len(args) < 2 {
		printUsage("", cmdMap)
	}

	cmd := args[1]

	if len(args) < 2 {
		printUsage(cmd, cmdMap)
	} else if _, exists := cmdMap[cmd]; !exists {
		printUsage(cmd, cmdMap)
	}

	options := make(map[string]string)

	for i := 2; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			var key, value string
			if strings.Contains(arg, "=") {
				parts := strings.SplitN(arg, "=", 2)
				key, value = parts[0], parts[1]
			} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				key, value = arg, args[i+1]
				i++
			} else {
				key = arg
				value = "true" // Handle boolean flags
			}
			options[key] = value
		}
	}

	return cmd, options, nil
}

// Validate the parsed options against the command definition
func validateOptions(command string, cmdMap map[string]any, options map[string]string) error {
	cmd := cmdMap[command]

	for optKey, opt := range cmd.Options {
		fmt.Sprintf("key is %s", optKey)
		// Check required options
		if opt.Required {
			if _, exists := options[opt.Short]; !exists && !existsInLong(options, opt.Long) {
				return fmt.Errorf("missing required option: %s (%s)", opt.Short, opt.Long)
			}
		}

		// Check valid choices
		if len(opt.Choices) > 0 {
			val, exists := options[opt.Short]
			if !exists {
				val = options[opt.Long]
			}
			if val != "" && !contains(opt.Choices, val) {
				return fmt.Errorf("invalid value for %s: %s, allowed: %v", opt.Short, val, opt.Choices)
			}
		}
	}

	return nil
}

// Check if a key exists in the long format options
func existsInLong(options map[string]string, long string) bool {
	for k := range options {
		if k == long {
			return true
		}
	}
	return false
}

// Utility function to check if a value exists in a slice
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// Print usage for a specific command
func printUsage(cmd string, cmdMap map[string]any) {

	_, exists := cmdMap[cmd]

	if !exists && (cmd != "-h" && cmd != "--help") {
		fmt.Printf("Unknown command %s\n", cmd)
		return
	}

	fmt.Println("Usage:")
	for cmdKey, cmdOptions := range cmdMap {
		fmt.Printf("  %s\t\n", cmdKey)
		var maxShort, maxLong int
		for _, cmdOption := range cmdOptions.(map[string]interface{}) {
			cmdOptionLong := cmdOption.(*options.Option).Long
			cmdOptionShort := cmdOption.(*options.Option).Short
			cmdOptionRequired := cmdOption.(*options.Option).Required
			isRequired := ""
			if cmdOptionRequired {
				isRequired = "(Required)"
			}
			if len(cmdOptionShort) > maxShort {
				maxShort = len(cmdOptionShort)
			}
			if len(cmdOptionLong) > maxLong {
				maxLong = len(cmdOptionLong)
			}
			format := fmt.Sprintf("  %%-%ds  %%-%ds  %%s %%s\n", maxShort, maxLong)
			cmdOptionHelp := cmdOption.(*options.Option).Help
			fmt.Printf(format, cmdOptionShort, cmdOptionLong, cmdOptionHelp, isRequired)
		}
	}
}

// Entry point
func MakeCLIFromAnsiblePlaybook() {

	// First positional parameter is the path to the playbook
	var playbook string = os.Args[1]
	if strings.HasSuffix(playbook, ".yaml") || strings.HasSuffix(playbook, ".yml") {
		// Remove the first positional parameter by re-slicing os.Args
		if strings.HasPrefix(playbook, "~/") {
			dirname, _ := os.UserHomeDir()
			playbook = filepath.Join(dirname, playbook[2:])
		}
		os.Args = append(os.Args[:1], os.Args[2:]...)
	} else {
		playbook = "Taskfile.yaml"
	}

	// Read the YAML configuration file
	data, err := ioutil.ReadFile(playbook)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var config []Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	commandsObj, commandsObjExists := config[0].Vars["commands"]
	globalOptionsObj, globalOptionsObjExists := config[0].Vars["globals"]
	var globalOptionsObjAttributes = make(map[string]any)
	var cliCommands = commandsObj.(map[string]any)
	if !commandsObjExists {
		logger.Fatal("Invalid playbook structure - no commands key found")
	}
	logger.Debug(globalOptionsObj, globalOptionsObjExists)
	var cmdStrings = []string{""}
	var cmdMap = make(map[string]any)
	for cmdObjName, cmdObj := range cliCommands {

		cmdString := cmdObjName
		cmdStrings = strings.Split(cmdString, "|")
		for _, cmdName := range cmdStrings {
			commandsObjAttributes := cmdObj.(map[string]any)
			if globalOptionsObjExists {
				globalOptionsObjAttributes = globalOptionsObj.(map[string]any)

			} else {
				globalOptionsObjAttributes = make(map[string]any)
			}
			cmdMap[cmdName] = ParseCmdOptions(cmdName, commandsObjAttributes, globalOptionsObjAttributes)
		}
	}

	command, options, err := parseArgs(cmdMap, os.Args)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Error:", err))
	}

	if _, exists := options["--help"]; exists {
		printUsage(command, cmdMap)
		return
	}

	err = validateOptions(command, cmdMap, options)
	if err != nil {
		fmt.Println("Validation Error:", err)
		os.Exit(1)
	}

	fmt.Printf("Command: %s\n", command)
	fmt.Printf("Options: %+v\n", options)
	fmt.Println("Command executed successfully!")
}
