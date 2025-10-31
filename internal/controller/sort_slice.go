package controller

import "slices"

func sortSlice(slice []string) []string {
	slices.Sort(slice)
	return slices.Compact(slice)
}
