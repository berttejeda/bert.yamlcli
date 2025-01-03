package ansible

import (
	"fmt"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Parse command-line arguments
func parseArgs(cmdMap map[string]any, args []string) (string, map[string]string, error) {

	if len(args) < 2 {
		printUsage("", cmdMap)
	}

	var cmd string = ""

	if len(args) > 1 {
		cmd = args[1]
	}

	if cmd == "--help" || cmd == "" {
		printUsage(cmd, cmdMap)
		os.Exit(0)
	}

	if len(args) < 2 {
		printUsage(cmd, cmdMap)
	} else if _, exists := cmdMap[cmd]; !exists {
		printUsage(cmd, cmdMap)
	}

	options := make(map[string]string)

	pattern := `^[\W]+[\w]`
	re := regexp.MustCompile(pattern)
	for i := 2; i < len(args); i++ {
		arg := args[i]
		if re.MatchString(arg) {
			var key, value string
			if strings.Contains(arg, "=") {
				parts := strings.SplitN(arg, "=", 2)
				key, value = parts[0], parts[1]
			} else if i+1 < len(args) && re.MatchString(arg) {
				key, value = arg, args[i+1]
				i++
			} else {
				key = arg
				value = "true" // Handle boolean flags
			}
			for k, v := range cmdMap[cmd].(map[string]interface{}) {
				// Define the regular expression
				pattern := fmt.Sprintf("^%s$|^%s$|^--help$", regexp.QuoteMeta(v.(*Option).Short), regexp.QuoteMeta(v.(*Option).Long))
				regex := regexp.MustCompile(pattern)
				// Check if the string matches the regex
				if regex.MatchString(key) {
					options[key] = k + "=" + value
				}
			}
		}
	}

	return cmd, options, nil
}

type Config struct {
	Hosts string         `yaml:"hosts"`
	Vars  map[string]any `yaml:"vars"`
}

// Validate the parsed options against the command definition
func validateOptions(command string, cmdMap map[string]any, cliArgs map[string]string) error {

	for cmdKey, cmdOptions := range cmdMap {
		logger.Debug(cmdKey)
		for _, cmdOption := range cmdOptions.(map[string]any) {
			cmdOptionLong := cmdOption.(*Option).Long
			cmdOptionShort := cmdOption.(*Option).Short
			cmdOptionRequired := cmdOption.(*Option).Required
			cmdOptionValidChoices := cmdOption.(*Option).Choices
			cliArgProvided := checkCLIArgProvided(cliArgs, cmdOptionLong, cmdOptionShort)
			if cmdOptionRequired {
				if !cliArgProvided {
					printUsage(command, cmdMap)
					if cmdOptionLong != "" && cmdOptionShort != "" {
						errMsg := fmt.Sprintf("Missing required option: %s|%s, see usage above", cmdOptionLong, cmdOptionShort)
						return fmt.Errorf(errMsg)
					} else if cmdOptionLong != "" {
						errMsg := fmt.Sprintf("Missing required option: %s, see usage above", cmdOptionLong)
						return fmt.Errorf(errMsg)
					} else {
						errMsg := fmt.Sprintf("Missing required option: %s, see usage above", cmdOptionShort)
						return fmt.Errorf(errMsg)
					}
				}
			}
			if len(cmdOptionValidChoices) > 0 {
				_, cliChoicesProvided, err := checkCLIArgChoices(cliArgs, cmdOptionValidChoices, cmdOptionLong, cmdOptionShort)
				if !cliChoicesProvided && err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Check if a key exists in the long format options
func checkCLIArgProvided(cliArgs map[string]string, long string, short string) bool {
	_, longCLIArgProvided := cliArgs[long]
	_, shortCLIArgProvided := cliArgs[short]
	if longCLIArgProvided || shortCLIArgProvided {
		return true
	}
	return false
}

func formatChoices(x interface{}) string {
	var validChoices = "["
	for _, choice := range x.([]interface{}) {
		switch choice.(type) {
		case int:
			validChoices += " " + strconv.Itoa(choice.(int))
		case string:
			validChoices += " " + choice.(string)
		}
	}
	validChoices += "]"
	return validChoices
}

// Check if a key exists in the long format options
func checkCLIArgChoices(cliArgs map[string]string, validChoices []any, long string, short string) ([]any, bool, error) {
	longCLIArg, longCLIArgProvided := cliArgs[long]
	shortCLIArg, shortCLIArgProvided := cliArgs[short]
	var choices = []any{}
	if longCLIArgProvided {
		choices = []any{longCLIArg}
	} else if shortCLIArgProvided {
		choices = []any{shortCLIArg}
	}
	for _, choice := range choices {
		// Check whether or not the choice provided from the command-line is present in the list of valid choices
		validChoice, err := containsChoice(validChoices, choice.(string))
		if err != nil {
			return []any{}, false, err
		} else if !validChoice {
			err := fmt.Errorf("invalid value for %s|%s (%s), allowed - %s", long, short, choice.(string), formatChoices(validChoices))
			return []any{}, false, err
		}
	}

	return []any{}, true, nil

}

// Print usage for a specific command
func printUsage(cmd string, cmdMap map[string]any) {

	_, exists := cmdMap[cmd]

	if !exists && (cmd != "-h" && cmd != "--help") && cmd != "" {
		fmt.Printf("Unknown command %s\n", cmd)
		return
	}

	fmt.Println("Usage:")
	for cmdKey, cmdOptions := range cmdMap {
		if cmd != "" {
			if cmdKey != cmd {
				continue
			}
		}
		fmt.Printf("  %s\t\n", cmdKey)
		var maxShort, maxLong int
		for _, cmdOption := range cmdOptions.(map[string]any) {
			cmdOptionLong := cmdOption.(*Option).Long
			cmdOptionShort := cmdOption.(*Option).Short
			cmdOptionRequired := cmdOption.(*Option).Required
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
			cmdOptionHelp := cmdOption.(*Option).Help
			fmt.Printf(format, cmdOptionShort, cmdOptionLong, cmdOptionHelp, isRequired)
		}
	}
}

// Entry point
func MakeCLIFromAnsiblePlaybook(playbook string, args []string) (string, map[string]string, string, []KeyValue, string) {

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

	command, cliArgs, err := parseArgs(cmdMap, args)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Error:", err))
	}

	if _, exists := cliArgs["--help"]; exists {
		printUsage(command, cmdMap)
		os.Exit(0)
	}

	err = validateOptions(command, cmdMap, cliArgs)
	if err != nil {
		fmt.Println("Validation Error:", err)
		os.Exit(1)
	}

	ansibleScriptArgs := map[string]interface{}{
		"command":  command,
		"playbook": playbook,
	}
	ansibleScript, ansibleScriptOptions, ansibleScriptWrapperFile := MakeAnsibleScript(ansibleScriptArgs, config, cliArgs)
	return command, cliArgs, ansibleScript, ansibleScriptOptions, ansibleScriptWrapperFile
}
