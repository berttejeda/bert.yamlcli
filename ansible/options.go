package ansible

import (
	"fmt"
	logger "github.com/sirupsen/logrus"
)

type HelpOption struct {
	Message  string
	Example  string
	Examples []any
}

type Option struct {
	Help           string
	OptionsSpacing int
	Short          string
	Long           string
	Variable       string
	Required       bool
	TypeOf         string
	Choices        []any
	Source         string
	Shell          string
}

func ParseCmdOptions(cmdName string, commandsObjAttributes map[string]any, globalOptionsObjAttributes map[string]any) (map[string]any, map[string]any) {

	optionsMap := make(map[string]map[string]any)
	cmdOptions := make(map[string]any)
	cmdOptionsHelp := make(map[string]any)
	// Define global options
	for globalObjAttributeName, globalOptionsObjAttributeData := range globalOptionsObjAttributes {

		attributeName := globalObjAttributeName
		attributeData := globalOptionsObjAttributeData.(map[string]any)
		switch attributeName {
		case "options":
			for optionName, optionData := range attributeData {
				options := optionData.(map[string]any)
				for optionAttributeName, optionAttributeData := range options {
					optionKey := optionName
					optionDataKey := optionAttributeName
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

	}

	// Define per-command options
	for cmdOptionName, cmdOptionData := range commandsObjAttributes {

		attributeName := cmdOptionName
		attributeData := cmdOptionData.(map[string]any)
		switch attributeName {
		case "options":
			for optionName, optionData := range attributeData {
				options := optionData.(map[string]any)
				for optionAttributeName, optionAttributeData := range options {
					optionKey := optionName
					optionDataKey := optionAttributeName
					if _, ok := optionsMap[optionKey]; !ok {
						optionsMap[optionKey] = map[string]any{}
					}
					if _, ok := optionsMap[optionKey][optionDataKey]; !ok {
						optionsMap[optionKey][optionDataKey] = optionAttributeData
					}
				}
			}
		case "help":
			helpObj := new(HelpOption)
			helpMessage, helpMessageExists := attributeData["message"]
			if helpMessageExists {
				helpObj.Message = helpMessage.(string)
			}
			helpExample, helpExampleExists := attributeData["example"]
			if helpExampleExists {
				helpObj.Example = helpExample.(string)
			}
			helpExamples, helpExamplesExists := attributeData["examples"]
			if helpExamplesExists {
				helpObj.Examples = helpExamples.([]any)
			}
			cmdOptionsHelp[cmdName] = helpObj
		default:
			continue
		}

		// Read options data for each command
		for optionKey, optionData := range optionsMap {
			// Initialize the Option Object
			optionObj := new(Option)
			// Long command-line flag
			optionLong, optionLongExists := optionData["long"].(string)
			logger.Debug(optionLong)
			if !optionLongExists {
				logger.Debug(fmt.Sprintf("%s.'%s' - no long option exists for this flag", cmdName, optionKey))
			}
			longOption := optionLong
			optionObj.Long = longOption
			// Short command-line flag
			short, shortExists := optionData["short"].(string)
			logger.Debug(short)
			if !shortExists {
				logger.Debug(fmt.Sprintf("'%s.%s' - no short option exists for this flag", cmdName, optionKey))
			}
			shortOption := short
			optionObj.Short = shortOption
			// Skip the option if neither the short nor long command-line flags are defined
			if longOption == "" && shortOption == "" {
				logger.Error(fmt.Sprintf("Skipping '%s.%s', as no long or short option exists for this flag", cmdName, optionKey))
				continue
			}
			// Option Type
			optionType, optionTypePresent := optionData["type"].(string)
			// Skip the option if its type is undefined
			if !optionTypePresent {
				logger.Error(fmt.Sprintf("Skipping '%s.%s' as no type data exists for this flag", cmdName, optionKey))
				continue
			}
			optionObj.TypeOf = optionType
			// Function types
			if optionType == "function" {
				// Option Shell
				optionShell, optionShellPresent := optionData["shell"].(string)
				if optionShellPresent {
					optionObj.Shell = optionShell
				}
				// Option Source
				optionSource, optionSourcePresent := optionData["source"].(string)
				if !optionSourcePresent {
					logger.Error(fmt.Sprintf("Skipping '%s.%s' as this is a function type with no source defined", cmdName, optionKey))
					continue
				} else {
					optionObj.Source = optionSource
				}
			}
			// Option Help
			optionHelp, optionHelpPresent := optionData["help"].(string)
			if !optionHelpPresent {
				optionHelp = ""
			}
			optionObj.Help = optionHelp
			// Check if option is Required
			optionRequired, optionRequiredPresent := optionData["required"].(bool)
			if !optionRequiredPresent {
				optionObj.Required = false
			} else {
				optionObj.Required = optionRequired
			}
			// Option Choices
			optionChoices, optionChoicesPresent := optionData["choices"].([]any)
			if optionChoicesPresent {
				optionObj.Choices = optionChoices
			}
			// Populate the corresponding cmdOptions key
			optionObj.OptionsSpacing = len(longOption + shortOption)
			if longOption == "--ansible-connection-password" {
				logger.Debug("ok")
			}
			cmdOptions[optionKey] = optionObj
		}
	}

	return cmdOptions, cmdOptionsHelp

}
