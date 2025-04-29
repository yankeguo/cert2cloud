package main

import (
	"maps"
	"slices"
	"strings"
)

func cleanJoined(vals []string) string {
	m := map[string]struct{}{}
	for _, domain := range vals {
		m[domain] = struct{}{}
	}
	vals = slices.Collect(maps.Keys(m))
	slices.Sort(vals)
	return strings.Join(vals, ",")
}

func cleanJoinedPtr(vals []*string) string {
	var dVals []string
	for _, val := range vals {
		dVals = append(dVals, *val)
	}
	return cleanJoined(dVals)
}
