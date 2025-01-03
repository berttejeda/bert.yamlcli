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
			err := fmt.Errorf("Invalid type for %v, supported types are string, int", choice)
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
