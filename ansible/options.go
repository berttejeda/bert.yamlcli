package ansible

import (
	"fmt"
	logger "github.com/sirupsen/logrus"
)

type Option struct {
	Help     string
	Short    string
	Long     string
	Variable string
	Required bool
	TypeOf   string
}

func ParseCmdOptions(cmdName string, commandsObjAttributes map[string]any, globalOptionsObjAttributes map[string]any) map[string]any {

	optionsMap := make(map[string]map[string]any)
	cmdOptions := make(map[string]any)

	for globalObjAttributeName, globalOptionsObjAttributData := range globalOptionsObjAttributes {

		attributeName := globalObjAttributeName
		attributeData := globalOptionsObjAttributData.(map[string]any)
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
		default:
			continue
		}

		for optionKey, optionData := range optionsMap {
			optionObj := new(Option)
			logger.Debug(optionKey, optionData)
			optionLong, optionLongExists := optionData["long"].(string)
			logger.Debug(optionLong)
			if !optionLongExists {
				logger.Error(fmt.Sprintf("Skipping '%s' as no long option exists for this flag", optionKey))
				continue
			}
			longOption := optionLong
			optionObj.Long = longOption
			short, shortExists := optionData["short"].(string)
			logger.Debug(short)
			if !shortExists {
				logger.Error(fmt.Sprintf("Skipping '%s' as no short option exists for this flag", optionKey))
				continue
			}
			shortOption := short
			optionObj.Short = shortOption
			optionType, optionTypePresent := optionData["type"].(string)
			if !optionTypePresent {
				logger.Error(fmt.Sprintf("Skipping '%s' as no type data exists for this flag", optionKey))
				continue
			}
			optionObj.TypeOf = optionType
			optionHelp, optionHelpPresent := optionData["help"].(string)
			if !optionHelpPresent {
				optionHelp = ""
			}
			optionObj.Help = optionHelp

			optionRequired, optionRequiredPresent := optionData["required"].(bool)
			if !optionRequiredPresent {
				optionObj.Required = false
			} else {
				optionObj.Required = optionRequired
			}
			optionObj.Help = optionHelp
			cmdOptions[optionKey] = optionObj
		}
	}

	return cmdOptions

}
