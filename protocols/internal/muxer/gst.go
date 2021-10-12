package muxer

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0 glib-2.0
#include "gst.h"
*/
import "C"
import (
	"unsafe"

	"github.com/mattn/go-pointer"
)

func init() {
	C.gstreamer_init()
}

type Pipeline struct {
	Pipeline *C.GstElement

	OnBuffer func(buffer []byte)
	OnRtcp func()
}

func NewPipeline() *Pipeline {
	p := &Pipeline{}
	p.Pipeline = C.gstreamer_run(pointer.Save(p))
	return p
}

//export goHandlePipelineBuffer
func goHandlePipelineBuffer(buffer unsafe.Pointer, bufferLen C.int, duration C.int, data unsafe.Pointer) {
	pipeline := pointer.Restore(data).(*Pipeline)
	if pipeline.OnBuffer != nil {
		pipeline.OnBuffer(C.GoBytes(buffer, bufferLen))
	}
}

//export goHandlePipelineRtcp
func goHandlePipelineRtcp(buffer unsafe.Pointer, bufferLen C.int, duration C.int, data unsafe.Pointer) {
	_ = pointer.Restore(data).(*Pipeline)
}
