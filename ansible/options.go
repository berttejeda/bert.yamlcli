package ansible

import (
	"github.com/alecthomas/kingpin/v2"
	logger "github.com/sirupsen/logrus"
	"log"
	"strconv"
)

func mergeOptions(commandOptions, commandOptionsGlobal map[any]any) map[any]any {
	result := make(map[any]any)

	// Copy all key-value pairs from commandOptions to result
	for key, value := range commandOptions {
		result[key] = value
	}

	// Copy all key-value pairs from commandOptionsGlobal["options"] to result["options"]
	for _, globalOptions := range commandOptionsGlobal {
		for globalOptionKey, globalOptionValue := range globalOptions.(map[any]any) {
			for _, commndOptionsAttributes := range result {
				for optionsAttributeKey, optionsAttributeValue := range commndOptionsAttributes.(map[any]any) {
					if optionsAttributeKey.(string) == "options" {
						optionsAttributeValue.(map[any]any)[globalOptionKey.(string)] = globalOptionValue
					}
				}

			}
		}
	}

	return result
}

func parseOptions(app *kingpin.Application, commandsObj any, commandsObjExists bool, ExistingCLIArguments map[string]any, globalOptionsObjExists bool, globalOptionsObj any) map[string]any {

	var commands map[any]any
	if commandsObjExists {
		if globalOptionsObjExists {
			commands = mergeOptions(commandsObj.(map[any]any), globalOptionsObj.(map[any]any))
		} else {
			commands = commandsObj.(map[any]any)
		}
	} else {
		log.Fatalf("Failed to parse config file: [0].vars.commands key missing")
	}

	cliArguments := ExistingCLIArguments

	for cmdObjName, cmdObj := range commands {

		optionsMap := make(map[string]map[string]any)
		cmdName := cmdObjName.(string)
		cmd := app.Command(cmdName, "")
		cmdObjAttributes := cmdObj.(map[any]any)

		for cmdObjAttributeName, cmdObjAttributeData := range cmdObjAttributes {

			attributeName := cmdObjAttributeName.(string)
			attributeData := cmdObjAttributeData.(map[any]any)

			switch attributeName {
			case "options":
				for optionName, optionData := range attributeData {
					options := optionData.(map[any]any)
					for optionAttributeName, optionAttributeData := range options {
						optionKey := optionName.(string)
						optionDataKey := optionAttributeName.(string)
						if _, ok := optionsMap[optionKey]; !ok {
							optionsMap[optionKey] = map[string]any{}
						}
						if _, ok := optionsMap[optionKey][optionDataKey]; !ok {
							optionsMap[optionKey][optionDataKey] = optionAttributeData
						}
					}
				}
			default:
				continue
			}

			for optionKey, optionData := range optionsMap {
				logger.Debug(optionKey, optionData)
				optionLong := optionData["long"].(string)
				short := optionData["short"].(string)
				optionType := optionData["type"].(string)
				optionHelp := optionData["help"].(string)
				optionAllowMultipleValue, optionAllowMultiplePresent := optionData["allow_multiple"]
				optionAllowMultiple := optionAllowMultiplePresent && optionAllowMultipleValue.(bool)
				optionChoicesValue, optionChoicesPresent := optionData["choices"]
				optionIsRequired, optionHasRequired := optionData["required"]
				optionDefault, optionHasDefault := optionData["default"]

				var optionHasChoices bool
				if optionChoicesPresent {
					optionHasChoices = len(optionChoicesValue.([]any)) > 0 && optionChoicesPresent
				} else {
					optionHasChoices = false
				}

				//_, optionFromEnvVariable := optionData["env_var"]
				flag := cmd.Flag(optionLong, optionHelp).Short(rune(short[0]))

				switch optionType {
				case "string":
					if optionAllowMultiple {
						cliArguments[optionKey] = flag.Strings()
					} else {
						cliArguments[optionKey] = flag.String()
					}
				case "bool":
					cliArguments[optionKey] = flag.Bool()
				case "int":
					if optionAllowMultiple {
						cliArguments[optionKey] = flag.Int()
					} else {
						cliArguments[optionKey] = flag.Ints()
					}
				}

				if optionHasChoices {
					validChoicesData, _ := optionData["choices"]
					var validChoices []string
					for _, choice := range validChoicesData.([]any) {
						switch choice.(type) {
						case string:
							validChoices = append(validChoices, choice.(string))
						case int:
							value := strconv.Itoa(choice.(int))
							validChoices = append(validChoices, value)
						}
					}
					flag.Enum(validChoices...)
				}

				if optionHasDefault {
					switch optionDefault.(type) {
					case string:
						flag.Default(optionDefault.(string))
					case int:
						value := strconv.Itoa(optionDefault.(int))
						flag.Default(value)
					}
				}

				if optionHasRequired && optionIsRequired.(bool) {
					flag.Required()
				}

			}
		}

	}

	return cliArguments

}
