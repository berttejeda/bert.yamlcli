package ansible

import (
	"bytes"
	"encoding/json"
	"fmt"
	logger "github.com/sirupsen/logrus"
	"log"
	"reflect"
	"strconv"
	"strings"
	"text/template"
)

// Convert map to slice of key-value pairs
type KeyValue struct {
	Key   string
	Value any
}

// Recursive function to convert map[interface{}]interface{} to map[string]interface{}
func convertMap(m interface{}) interface{} {
	switch v := m.(type) {
	case map[interface{}]interface{}:
		convertedMap := make(map[string]interface{})
		for key, value := range v {
			strKey := fmt.Sprintf("%v", key)         // Convert key to string
			convertedMap[strKey] = convertMap(value) // Recursively handle nested maps
		}
		return convertedMap
	case []interface{}:
		for i, elem := range v {
			v[i] = convertMap(elem) // Recursively handle slices
		}
		return v
	default:
		return v // Return the value as is (e.g., for primitive types like string, int)
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

func MakeAnsibleScript(args map[string]interface{}, config []Config, cliArguments map[string]string) (string, []KeyValue, string) {

	funcs := map[string]any{
		"contains":      strings.Contains,
		"convertMap":    convertMap,
		"getType":       reflect.TypeOf,
		"hasPrefix":     strings.HasPrefix,
		"hasSuffix":     strings.HasSuffix,
		"sanitizeInput": sanitizeInput,
		"nullString": func(input string) (string, error) {
			return "", nil
		},
	}

	tpl := `#!/usr/bin/env bash
# ANSI Colors
RESTORE=$(echo -en '')
RED=$(echo -en '')
GREEN=$(echo -en '')
YELLOW=$(echo -en '')
BLUE=$(echo -en '')
MAGENTA=$(echo -en '')
PURPLE=$(echo -en '')
CYAN=$(echo -en '')
LIGHTGRAY=$(echo -en '')

{{- range . }}
# Ordered Variables
{{- $valueType := getType .Value }}
{{- $isAnsibleEnvVariable := hasPrefix .Key "ANSIBLE_" }}
{{- if $isAnsibleEnvVariable }}
export {{ .Key }}={{ sanitizeInput .Value }}
{{- else }}
{{ .Key }}={{ sanitizeInput .Value }}
{{- end }}

{{- if eq .Key "pre_execution" }}
# Pre-Execution
{{ .Value }}
{{- end }}

{{- end }}

# Ansible Command(s)
ansible-playbook \
${__ansible_run_flags__} \
-i "${__inventory__}" \
{{- range . }}
-e "{'{{ .Key }}':'${{ .Key }}'}" \
{{- end }}
${__playbook__} -l ${playbook_targets}
`
	logger.Debug(cliArguments)
	var ansibleScriptWrapperFile string = ""
	var ansibleScriptOptions []KeyValue
	var existingKeys []string
	ansibleScriptOptions = append(ansibleScriptOptions, KeyValue{"__command__", args["command"]})
	ansibleScriptOptions = append(ansibleScriptOptions, KeyValue{"__playbook__", args["playbook"]})
	for _, v := range cliArguments {
		// Split only if "=" exists
		index := strings.Index(v, "=")
		if index != -1 {
			key := v[:index]     // Substring before the "="
			value := v[index+1:] // Substring after the "="
			if strings.HasPrefix(value, "=") {
				value = strings.TrimPrefix(value, "=")
			}
			existingKeys = append(existingKeys, key)
			if key == "__wrapper_file__" {
				ansibleScriptWrapperFile = value
			} else {
				ansibleScriptOptions = append(ansibleScriptOptions, KeyValue{key, value})
			}
		}
	}
	for k, v := range config[0].Vars {
		if k == "commands" || k == "globals" {
			continue
		} else if k == "__ordered_vars__" {
			for _, orderedVarValue := range v.([]interface{}) {
				for orderedSubVarKey, orderedSubVarValue := range orderedVarValue.(map[string]interface{}) {
					if orderedSubVarKey == "__wrapper_file__" && ansibleScriptWrapperFile == "" {
						ansibleScriptWrapperFile = orderedSubVarValue.(string)
					}
					if StringArrayContains(existingKeys, orderedSubVarKey) {
						continue
					} else {
						ansibleScriptOptions = append(ansibleScriptOptions, KeyValue{orderedSubVarKey, orderedSubVarValue})
					}
				}

			}
		}
	}

	tmpl := template.Must(template.New("AnsibleVars").Funcs(funcs).Parse(tpl))
	var tmplResult bytes.Buffer
	err := tmpl.Execute(&tmplResult, ansibleScriptOptions)
	if err != nil {
		log.Fatal(err)
	}
	return tmplResult.String(), ansibleScriptOptions, ansibleScriptWrapperFile
}
