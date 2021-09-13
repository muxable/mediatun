package rist

import (
	/*
		#cgo LDFLAGS: -lrist
		#include "rist.h"
	*/
	"C"
)
import "unsafe"

type DataBlock C.struct_rist_data_block

func NewDataBlock(data []byte) *DataBlock {
	return &DataBlock{
		payload:     C.CBytes(data),
		payload_len: C.ulong(len(data)),
	}
}

func (d *DataBlock) GetPayload() []byte {
	return C.GoBytes(d.payload, C.int(uint64(d.payload_len)))
}

// GetRawPayload gets the underlying CBytes.
func (d *DataBlock) GetRawPayload() (unsafe.Pointer, uint64) {
	return d.payload, uint64(d.payload_len)
}

func (d *DataBlock) GetRTPTimestamp() uint32 {
	return uint32((int64(d.ts_ntp) * 90000) >> 32)
}

func (d *DataBlock) GetFlowId() uint32 {
	return uint32(d.flow_id)
}

func (d *DataBlock) GetVirtSrcPort() uint16 {
	return uint16(d.virt_src_port)
}

func (d *DataBlock) GetVirtDstPort() uint16 {
	return uint16(d.virt_dst_port)
}
