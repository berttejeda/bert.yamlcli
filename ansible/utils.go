package ansible

import (
	"fmt"
	"strconv"
)

func containsChoice(arr []any, target string) (bool, error) {
	for _, choice := range arr {
		// Use type assertion to check if v is a string
		switch choice.(type) {
		case int:
			comparison, err := strconv.Atoi(target)
			if choice.(int) == comparison && err == nil {
				return true, nil
			}
		case string:
			if choice == target {
				return true, nil
			}
		default:
			err := fmt.Errorf("Invalid type for %v, supported types are string and int", choice)
			return false, err
		}
	}
	return false, nil
}

func StringArrayContains(slice []string, item string) bool {
	for _, str := range slice {
		if str == item {
			return true
		}
	}
	return false
}

func findIndex(arr []string, target string) int {
	for i, v := range arr {
		if v == target {
			return i // Return the index if the item is found
		}
	}
	return -1 // Return -1 if the item is not found
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
