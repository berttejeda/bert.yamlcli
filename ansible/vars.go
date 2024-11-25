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
			return v
		}
	case int:
		return strconv.Itoa(v)
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

func MakeCLIVarsFromAnsiblePlaybook(config []Config, cliArguments map[string]interface{}) map[string]any {

	funcs := map[string]any{
		"contains":      strings.Contains,
		"convertMap":    convertMap,
		"getType":       reflect.TypeOf,
		"hasPrefix":     strings.HasPrefix,
		"hasSuffix":     strings.HasSuffix,
		"sanitizeInput": sanitizeInput,
	}

	tpl := `
{{- range $k, $v := $.Vars }}
{{- $isAnsibleEnvVariable := hasPrefix $k "ANSIBLE_" }}
{{- $valueType := getType $v }}
{{- if $isAnsibleEnvVariable }}
export {{ $k }}={{ sanitizeInput $v -}}
{{- else }}
{{ $k }}={{ sanitizeInput $v }}
{{- end }}
{{- end }}
`
	logger.Debug(cliArguments)
	tmpl := template.Must(template.New("test").Funcs(funcs).Parse(tpl))
	var tmplResult bytes.Buffer
	err := tmpl.Execute(&tmplResult, config[0])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(tmplResult.String())
	return config[0].Vars
}
