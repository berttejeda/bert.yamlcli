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

func MakeAnsibleScript(args map[string]interface{}, config []Config, cliArguments map[string]string) string {

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

{{- if ne .Key "script_vars" }}
{{ .Key }}={{ .Value }}
{{- else }}
{{- range $nestedKey,$nestedValue := .Value }}
{{- range $key,$value := $nestedValue }}

{{- $isAnsibleEnvVariable := hasPrefix $key "ANSIBLE_" }}

{{- if $isAnsibleEnvVariable }}
export {{ $key }}={{ sanitizeInput $value }}
{{- else }}
{{ $key }}={{ sanitizeInput $value }}
{{- end }}

{{- end }}
{{- end }}
{{- end }}

{{- $valueType := getType .Value }}

{{- if eq .Key "script_vars" }}
{{- range $nestedKey,$nestedValue := .Value }}
{{- range $key,$value := $nestedValue }}
{{- if eq $key "pre_execution" }}
# Pre-Execution
{{ $value }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{- end }}

# Ansible Command(s)
ansible-playbook \
${__ansible_run_flags__} \
-i "${__inventory__}" \
{{- range . }}
{{- if ne .Key "script_vars" }}
-e "{'{{ .Key }}':'${{ .Key }}'}" \
{{- else }}
{{- range $nestedKey,$nestedValue := .Value }}
{{- range $key,$value := $nestedValue }}
-e "{'{{ $key }}':'${{ $key }}'}" \
{{- end }}
{{- end }}
{{- end }}
{{- end }}
${__playbook__} -l ${playbook_targets}

`
	logger.Debug(cliArguments)
	var pairs []KeyValue
	pairs = append(pairs, KeyValue{"__command__", args["command"]})
	//inventoryFile, inventoryFileSpecified := args["inventory_file"]
	//if inventoryFileSpecified {
	//	pairs = append(pairs, KeyValue{"_inventory_", inventoryFile})
	//}
	pairs = append(pairs, KeyValue{"__playbook__", args["playbook"]})
	for _, v := range cliArguments {
		// Split only if "=" exists
		index := strings.Index(v, "=")
		if index != -1 {
			key := v[:index]     // Substring before the "="
			value := v[index+1:] // Substring after the "="
			if strings.HasPrefix(value, "=") {
				value = strings.TrimPrefix(value, "=")
			}
			pairs = append(pairs, KeyValue{key, value})
		}
	}
	for k, v := range config[0].Vars {
		if k == "commands" || k == "globals" {
			continue
		} else {
			pairs = append(pairs, KeyValue{k, v})
		}
	}
	tmpl := template.Must(template.New("AnsibleVars").Funcs(funcs).Parse(tpl))
	var tmplResult bytes.Buffer
	err := tmpl.Execute(&tmplResult, pairs)
	if err != nil {
		log.Fatal(err)
	}
	return tmplResult.String()
}
