package ansible

import (
	"github.com/alecthomas/kingpin/v2"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

type Config struct {
	Hosts string         `yaml:"hosts"`
	Vars  map[string]any `yaml:"vars"`
}

func MakeCLIFromAnsiblePlaybook(configFile string) (string, map[string]any) {
	// Read the YAML configuration file
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var config []Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	// Create the application
	app := kingpin.New("MyApp", "My application description")
	app.Version("1.0.0")
	//// Add global options
	//for _, option := range config[0].Globals.Options {
	//	switch option.Type {
	//	case "choice":
	//		app.Flag(option.Long, option.Help).Short(rune(option.Short[1])).Enum(option.Options...)
	//	case "str":
	//		app.Flag(option.Long, option.Help).Short(rune(option.Short[1])).String()
	//	default:
	//		app.Flag(option.Long, option.Help).Short(rune(option.Short[1])).String()
	//	}
	//}
	//
	//// Add commands and their options

	commandsObj, commandsObjExists := config[0].Vars["commands"]
	var commands map[any]any
	if commandsObjExists {
		commands = commandsObj.(map[any]any)
	} else {
		log.Fatalf("Failed to parse config file: [0].vars.commands key missing")
	}

	cliArguments := make(map[string]any)

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

	cli := kingpin.MustParse(app.Parse(os.Args[1:]))

	return cli, cliArguments

}
