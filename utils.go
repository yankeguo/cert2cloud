package main

import (
	"maps"
	"slices"
	"strings"
	"time"
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

func timeDiff(t1, t2 time.Time) time.Duration {
	if t1.After(t2) {
		return t1.Sub(t2)
	}
	return t2.Sub(t1)
}
