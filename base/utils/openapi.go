package utils

import (
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"time"
)

// let's "define" all function on one place so it's easy to find them

var PtrBool = vmaas.PtrBool
var PtrInt = vmaas.PtrInt
var PtrInt32 = vmaas.PtrInt32
var PtrInt64 = vmaas.PtrInt64
var PtrFloat32 = vmaas.PtrFloat32
var PtrFloat64 = vmaas.PtrFloat64
var PtrString = vmaas.PtrString
var PtrTime = vmaas.PtrTime

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
