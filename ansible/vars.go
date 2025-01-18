package ansible

import (
	logger "github.com/sirupsen/logrus"
	"log"
	"strings"
)

func MakeAnsibleScript(args map[string]interface{}, config []Config, cliArguments map[string]string) (string, []KeyValue, string) {

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
{{- $valueType := getType .Value }}
{{- $isAnsibleEnvVariable := hasPrefix .Key "ANSIBLE_" }}
{{- if $isAnsibleEnvVariable }}
# Ordered Variables
export {{ .Key }}={{ sanitizeInput .Value }}
{{- else }}
{{- if and (ne .Key "__functions__") (ne .Key "__pre_execution__") }}
# Ordered Variables
{{ .Key }}={{ sanitizeInput .Value }}
{{- end }}
{{- end }}
{{- if eq .Key "__functions__" }}
# Functions
{{ .Value }}
{{- end }}
{{- if eq .Key "__pre_execution__" }}
# Pre-Execution
{{ .Value }}
{{- end }}
{{- end }}
# Ansible Command(s)
ansible-playbook \
${__ansible_args_raw__} \
${__ansible_args_extra__} \
-i "${__inventory__}" \
{{- range . }}
{{- if or (eq .Key "__functions__") (eq .Key "__pre_execution__") }}
{{- continue }}
{{- else }}
-e "{'{{ .Key }}':'${{ .Key }}'}" \
{{- end }}
{{- end }}
${__playbook__} -l ${playbook_targets}
`
	logger.Debug(cliArguments)
	var ansibleScriptWrapperFile string = ""
	var ansibleScriptOptions []KeyValue
	var existingKeys []string
	var functionsBlock string
	var __pre_execution__ interface{}
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
		if k == "__ordered_vars__" {
			for _, orderedVarValue := range v.([]interface{}) {
				for orderedSubVarKey, orderedSubVarValue := range orderedVarValue.(map[string]interface{}) {
					if orderedSubVarKey == "__wrapper_file__" && ansibleScriptWrapperFile == "" {
						ansibleScriptWrapperFile = orderedSubVarValue.(string)
					}
					// Variables defined via CLIArgs take precedence over those present in the playbook
					if StringArrayContains(existingKeys, orderedSubVarKey) {
						continue
					} else {
						if orderedSubVarKey == "__pre_execution__" {
							__pre_execution__ = orderedSubVarValue
						} else {
							ansibleScriptOptions = append(ansibleScriptOptions, KeyValue{orderedSubVarKey, orderedSubVarValue})
						}
					}
				}

			}
		}
	}
	ansibleScriptOptions = append(ansibleScriptOptions, KeyValue{"__functions__", functionsBlock})
	ansibleScriptOptions = append(ansibleScriptOptions, KeyValue{"__pre_execution__", __pre_execution__})
	tmpl, err := templatizeArrayOfKeyVars(tpl, "AnsibleVars", ansibleScriptOptions)
	if err != nil {
		log.Fatal(err)
	}
	return tmpl, ansibleScriptOptions, ansibleScriptWrapperFile
}
