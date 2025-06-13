package ansible

import (
	"fmt"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Parse command-line arguments
func parseArgs(cmdMap map[string]any, cmdMapHelp map[string]any, args []string) (string, map[string]string, error) {

	// Initialize options map
	options := make(map[string]string)
	// Process Raw Args
	rawArgsDelimiterIndex := findIndex(args, "---")
	if rawArgsDelimiterIndex != -1 {
		logger.Debug(fmt.Sprintf("Found raw args Delimiter (---) at index %v", rawArgsDelimiterIndex))
		rawArgs := args[rawArgsDelimiterIndex+1:]
		logger.Debug(fmt.Sprintf("Raw args are %s", rawArgs))
		args = args[0:rawArgsDelimiterIndex]
		options["__ansible_args_raw__"] = "__ansible_args_raw__="
		for _, v := range rawArgs {
			options["__ansible_args_raw__"] += v + " "
		}
	}

	// Print Usage if not enough args passed in
	if len(args) < 2 {
		printUsage("", cmdMap, cmdMapHelp)
	}

	// Initialize command variable
	var cmd string
	if len(args) > 1 {
		cmd = args[1]
	}

	// Print Usage if --help specified
	if cmd == "--help" || cmd == "" {
		cmd = ""
		printUsage(cmd, cmdMap, cmdMapHelp)
		os.Exit(0)
	}

	// Print Usage if command specified, but not enough args passed in
	if len(args) < 2 {
		printUsage(cmd, cmdMap, cmdMapHelp)
	} else if _, exists := cmdMap[cmd]; !exists {
		printUsage(cmd, cmdMap, cmdMapHelp)
	}

	// Use a regular expression to match command-line flags/args
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
				if k == "__help__" {
					continue
				}
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
func validateOptions(command string, cmdMap map[string]any, cmdMapHelp map[string]any, cliArgs map[string]string) error {

	for _, cmdOptions := range cmdMap {
		for cmdKey, cmdOption := range cmdOptions.(map[string]any) {
			if command != "" {
				if cmdKey != command {
					continue
				}
			}
			cmdOptionLong := cmdOption.(*Option).Long
			cmdOptionShort := cmdOption.(*Option).Short
			cmdOptionRequired := cmdOption.(*Option).Required
			cmdOptionValidChoices := cmdOption.(*Option).Choices
			cliArgProvided := checkCLIArgProvided(cliArgs, cmdOptionLong, cmdOptionShort)
			if cmdOptionRequired {
				if !cliArgProvided {
					printUsage(command, cmdMap, cmdMapHelp)
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
	var choices []any
	if longCLIArgProvided {
		choices = []any{longCLIArg}
	} else if shortCLIArgProvided {
		choices = []any{shortCLIArg}
	}
	for _, choice := range choices {
		// Check whether the choice provided from the command-line is present in the list of valid choices
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
func printUsage(cmd string, cmdMap map[string]any, cmdMapHelp map[string]any) {

	_, exists := cmdMap[cmd]
	if !exists && (cmd != "-h" && cmd != "--help") && cmd != "" {
		fmt.Printf("Unknown command %s\n", cmd)
		return
	}

	cmdUsageTemplate := `{{- range . }}
{{- $Spacing := .Spacing }}
{{ .Command }}: {{ .Help }}
Options:
{{- range $key, $value := .CommandOptions }}
{{- if and (ne $value.Long  "") (ne $value.Short  "") }}
{{ padCLIOptions $Spacing $value.Long $value.Short $value.Help }}
{{- else if ne $value.Long  "" }}
{{ padCLIOptions $Spacing $value.Long "" $value.Help }}
{{- else if ne $value.Short  "" }}
{{ padCLIOptions $Spacing "" $value.Short $value.Help }}
{{- end }}
{{- end }} 
{{- end }}

`
	fmt.Println("Usage:")
	spacing := GetOptionsMaxLength(cmdMap)
	for cmdKey, cmdOptions := range cmdMap {
		var cmdUsageObjects []cmdUsageMap
		if cmd != "" {
			if cmdKey != cmd {
				continue
			}
		}
		helpVars := map[string]interface{}{
			"__command__": cmdKey,
		}
		var helpExample string
		var helpMessage string
		var cmdUsageExample string
		var err error
		helpMessageObj := cmdMapHelp[cmdKey+".help"]
		if helpMessageObj != nil {
			helpMessageTemplate := cmdMapHelp[cmdKey+".help"].(map[string]interface{})[cmdKey].(*HelpOption).Message
			helpMessage, err = templatizeMap(helpMessageTemplate, "helpMessage", helpVars)
			if err != nil {
				logger.Debug(fmt.Sprintf("%v", err))
			}
			helpExampleTemplate := helpMessageObj.(map[string]interface{})[cmdKey].(*HelpOption).Example
			helpExample, err = templatizeMap(helpExampleTemplate, "helpMessage", helpVars)
			if err != nil {
				logger.Warning(fmt.Sprintf("%v", err))
				helpExample = ""
			}
		} else {
			helpExample = ""
		}
		cmdUsageObjects = append(cmdUsageObjects, cmdUsageMap{cmdKey, helpMessage, helpExample, cmdOptions, spacing})
		cmdUsageExample, err = templatizeArrayOfCmdUsageMaps(cmdUsageTemplate, "cmdUsage", cmdUsageObjects)
		if err != nil {
			logger.Fatal(fmt.Sprintf("Failed to print command usage %v", err))
		}
		fmt.Println(cmdUsageExample)
	}
}

// MakeCLIFromAnsiblePlaybook Entry point
func MakeCLIFromAnsiblePlaybook(playbook string, args []string) (string, map[string]string, string, []KeyValue, string) {

	// Read the YAML configuration file
	data, err := os.ReadFile(playbook)
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
	var cmdStrings = []string{""}
	var cmdMap = make(map[string]any)
	var cmdMapHelp = make(map[string]any)
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
			cmdMap[cmdName], cmdMapHelp[cmdName+".help"] = ParseCmdOptions(cmdName, commandsObjAttributes, globalOptionsObjAttributes)
		}
	}

	command, cliArgs, err := parseArgs(cmdMap, cmdMapHelp, args)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Error: %s", err))
	}

	if _, exists := cliArgs["--help"]; exists {
		printUsage(command, cmdMap, cmdMapHelp)
		os.Exit(0)
	}

	err = validateOptions(command, cmdMap, cmdMapHelp, cliArgs)
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
