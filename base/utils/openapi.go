package utils

import (
	"time"
)

func PtrBool(v bool) *bool           { return &v }
func PtrInt(v int) *int              { return &v }
func PtrInt32(v int32) *int32        { return &v }
func PtrInt64(v int64) *int64        { return &v }
func PtrFloat32(v float32) *float32  { return &v }
func PtrFloat64(v float64) *float64  { return &v }
func PtrString(v string) *string     { return &v }
func PtrTime(v time.Time) *time.Time { return &v }

func PtrSliceString(v []string) *[]string {
	return &v
}

func PtrTimeParse(ts string) *time.Time {
	t, _ := time.Parse(time.RFC3339, ts)
	return &t
}

func PtrBoolNil() *bool {
	var b *bool
	return b
}
