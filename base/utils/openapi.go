package utils

import (
	"time"
)

func PtrSliceString(v []string) *[]string {
	return &v
}

func PtrTimeParse(ts string) *time.Time {
	t, _ := time.Parse(time.RFC3339, ts)
	return &t
}
