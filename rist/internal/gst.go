package internal

/*
#cgo CFLAGS: -I/usr/local/lib/x86_64-linux-gnu
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0 glib-2.0
#include "gst.h"
*/
import "C"
import (
	"time"
	"unsafe"

	"github.com/mattn/go-pointer"
)

func init() {
	go C.gstreamer_main_loop()
	C.gstreamer_init()
}

type Pipeline struct {
	gstElement *C.GstElement

	OnVP8Buffer func([]byte, time.Duration)
	OnOpusBuffer	func([]byte, time.Duration)
}


func (p *Pipeline) Start() {
	pipelineStr := C.CString(`
		appsrc name=appsrc ! tsdemux name=demux
		demux. ! queue ! decodebin ! vp8enc ! appsink name=vp8appsink
		demux. ! queue ! decodebin ! opusenc ! appsink name=opusappsink`)
	defer C.free(unsafe.Pointer(pipelineStr))

	p.gstElement = C.gstreamer_start(pipelineStr, pointer.Save(p))
}

// Write writes an mpegts packet to the pipeline.
func (p *Pipeline) Write(buffer []byte) error {
	b := C.CBytes(buffer)
	defer C.free(b)
	C.gstreamer_push(p.gstElement, b, C.int(len(buffer)))

	return nil
}

// Close terminates the pipeline.
func (p *Pipeline) Close() {
	C.gstreamer_stop(p.gstElement)
}

//export goHandleVP8AppSinkBuffer
func goHandleVP8AppSinkBuffer(buffer unsafe.Pointer, bufferLen C.int, duration C.int, data unsafe.Pointer) {
	pipeline := pointer.Restore(data).(*Pipeline)
	if pipeline.OnVP8Buffer != nil {
		pipeline.OnVP8Buffer(C.GoBytes(buffer, bufferLen), time.Duration(duration))
	}
}
//export goHandleOpusAppSinkBuffer
func goHandleOpusAppSinkBuffer(buffer unsafe.Pointer, bufferLen C.int, duration C.int, data unsafe.Pointer) {
	pipeline := pointer.Restore(data).(*Pipeline)
	if pipeline.OnOpusBuffer != nil {
		pipeline.OnOpusBuffer(C.GoBytes(buffer, bufferLen), time.Duration(duration))
	}
}
