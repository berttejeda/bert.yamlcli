package ansible

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/template"
)

type KeyValue struct {
	Key   string
	Value any
}

type cmdUsageMap struct {
	Command     string
	Help        string
	HelpExample string
	//HelpExamples   []string
	CommandOptions any
	Spacing        int
}

type HelpMessage struct {
	Length int
}

func (h *HelpMessage) SetLength() {
	h.Length = 2
}

func padCLIOptions(n int, l string, s string, h string) string {
	var rightPadding int
	var cliOption string
	if len(l) > 0 && len(s) > 0 {
		cliOption = fmt.Sprintf("%s|%s", l, s)
	} else if len(l) > 0 {
		cliOption = l
	} else {
		cliOption = s
	}
	cliOptionLength := len(cliOption)
	rightPadding = n - cliOptionLength
	paddedCLIOptions := fmt.Sprintf("%s%*s%s", cliOption, rightPadding, "", h)
	return paddedCLIOptions
}

func GetOptionsMaxLength(data map[string]any) int {

	// Find maximum widths for columns
	maxCommandLength := len("Command")
	//maxHelpMessageLength := len("HelpMessage")

	for _, cmdAttribute := range data {
		for _, attribute := range cmdAttribute.(map[string]any) {
			optionSpacing := attribute.(*Option).OptionsSpacing
			if optionSpacing > maxCommandLength {
				maxCommandLength = optionSpacing
			}

		}
	}
	return maxCommandLength + 2
}

func sanitizeHelpInput(value interface{}) string {
	switch v := value.(type) {
	case string:
		if strings.Contains(v, "\n") {
			return "$(cat <<EOF\n" + v + "\nEOF\n)"
		} else {
			jsonData, err := json.Marshal(strings.TrimRight(v, " \n\r"))
			if err != nil {
				fmt.Println("Error marshaling JSON:", err)
				return ""
			}
			return string(jsonData)
		}
	case int:
		return strings.TrimRight(strconv.Itoa(v), " \n\r")
	// Add whatever other types you need
	case []interface{}:
		jsonData, err := json.Marshal(v)
		if err != nil {
			fmt.Println("Error marshaling JSON:", err)
			return ""
		}
		return string(jsonData)
	case map[interface{}]interface{}:
		convertedMap := convertMap(v)
		// Marshal the converted map to JSON
		jsonData, err := json.Marshal(convertedMap)
		if err != nil {
			fmt.Println("Error marshaling JSON:", err)
			return ""
		}
		return "$(cat <<EOF\n" + string(jsonData) + "\nEOF\n)"
	case *Option:
		return ""
	default:
		return ""
	}
}

func sanitizeInput(value interface{}) string {
	switch v := value.(type) {
	case string:
		if strings.Contains(v, "\n") {
			return "$(cat <<EOF\n" + v + "\nEOF\n)"
		} else {
			jsonData, err := json.Marshal(strings.TrimRight(v, " \n\r"))
			if err != nil {
				fmt.Println("Error marshaling JSON:", err)
				return ""
			}
			return string(jsonData)
		}
	case int:
		return strings.TrimRight(strconv.Itoa(v), " \n\r")
	// Add whatever other types you need
	case []interface{}:
		jsonData, err := json.Marshal(v)
		if err != nil {
			fmt.Println("Error marshaling JSON:", err)
			return ""
		}
		return string(jsonData)
	case map[interface{}]interface{}:
		convertedMap := convertMap(v)
		// Marshal the converted map to JSON
		jsonData, err := json.Marshal(convertedMap)
		if err != nil {
			fmt.Println("Error marshaling JSON:", err)
			return ""
		}
		return "$(cat <<EOF\n" + string(jsonData) + "\nEOF\n)"
	default:
		return ""
	}
}

var funcs = map[string]any{
	"contains":          strings.Contains,
	"convertMap":        convertMap,
	"getType":           reflect.TypeOf,
	"hasPrefix":         strings.HasPrefix,
	"hasSuffix":         strings.HasSuffix,
	"sanitizeInput":     sanitizeInput,
	"padCLIOptions":     padCLIOptions,
	"sanitizeHelpInput": sanitizeHelpInput,
	"nullString": func(input string) (string, error) {
		return "", nil
	},
}

func templatizeMap(tmplInput string, templateName string, templateVars map[string]any) (string, error) {

	tmpl := template.Must(template.New(templateName).Funcs(funcs).Parse(tmplInput))
	var tmplResultBuffer bytes.Buffer
	err := tmpl.Execute(&tmplResultBuffer, templateVars)
	if err != nil {
		return "", fmt.Errorf("error executing template: %v", err)
	}
	tmplResult := tmplResultBuffer.String()
	return tmplResult, nil
}

func templatizeArrayOfKeyVars(tmplInput string, templateName string, templateVars []KeyValue) (string, error) {

	tmpl := template.Must(template.New(templateName).Funcs(funcs).Parse(tmplInput))
	var tmplResultBuffer bytes.Buffer
	err := tmpl.Execute(&tmplResultBuffer, templateVars)
	if err != nil {
		return "", fmt.Errorf("error executing template - %v", err)
	}
	tmplResult := tmplResultBuffer.String()
	return tmplResult, nil
}

func templatizeArrayOfCmdUsageMaps(tmplInput string, templateName string, templateVars []cmdUsageMap) (string, error) {

	tmpl := template.Must(template.New(templateName).Funcs(funcs).Parse(tmplInput))
	var tmplResultBuffer bytes.Buffer
	err := tmpl.Execute(&tmplResultBuffer, templateVars)
	if err != nil {
		return "", fmt.Errorf("error executing template - %v", err)
	}
	tmplResult := tmplResultBuffer.String()
	return tmplResult, nil
}
