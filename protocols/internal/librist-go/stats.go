package rist

import (
	/*
		#cgo LDFLAGS: -lrist
		#include "rist.h"
	*/
	"C"
)

type Stats C.struct_rist_stats

func (s *Stats) GetJsonString() string {
	return C.GoStringN(s.stats_json, C.int(s.json_size))
}