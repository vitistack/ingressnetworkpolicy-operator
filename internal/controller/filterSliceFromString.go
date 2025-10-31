package controller

import (
	"strings"
)

func filterSliceFromString(slice []string) []string {

	var filteredSlice []string

	for _, item := range slice {
		if item != "" {
			filteredSlice = append(filteredSlice, strings.TrimSpace(item))
		}
	}

	return filteredSlice
}
