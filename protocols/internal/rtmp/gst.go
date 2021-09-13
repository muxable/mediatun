// Package gst provides an easy API to create an appsrc pipeline
package rtmp

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0
#include "gst.h"
*/
import "C"
import (
	"unsafe"
)

// StartMainLoop starts GLib's main loop
// It needs to be called from the process' main thread
// Because many gstreamer plugins require access to the main thread
// See: https://golang.org/pkg/runtime/#LockOSThread
func StartMainLoop() {
	C.gstreamer_receive_start_mainloop()
}

// Pipeline is a wrapper for a GStreamer Pipeline
type Pipeline struct {
	Pipeline *C.GstElement
}

// CreatePipeline creates a GStreamer Pipeline
func CreatePipeline(sink string) *Pipeline {
	pipelineStr := `flvmux name=mux ! ` + sink + `
	appsrc name=videosrc is-live=true format=time do-timestamp=true ! application/x-rtp, encoding-name=VP8-DRAFT-IETF-01 ! rtpvp8depay ! decodebin ! x264enc speed-preset=ultrafast tune=zerolatency key-int-max=20 ! video/x-h264,stream-format=byte-stream ! mux.
	appsrc name=audiosrc is-live=true format=time do-timestamp=true ! application/x-rtp, payload=96, encoding-name=OPUS ! rtpopusdepay ! decodebin ! audioresample ! audioconvert ! avenc_aac ! mux.`
	pipelineStrUnsafe := C.CString(pipelineStr)
	defer C.free(unsafe.Pointer(pipelineStrUnsafe))
	return &Pipeline{Pipeline: C.gstreamer_receive_create_pipeline(pipelineStrUnsafe)}
}

// Start starts the GStreamer Pipeline
func (p *Pipeline) Start() {
	C.gstreamer_receive_start_pipeline(p.Pipeline)
}

// Stop stops the GStreamer Pipeline
func (p *Pipeline) Stop() {
	C.gstreamer_receive_stop_pipeline(p.Pipeline)
}

// Push pushes a buffer on the appsrc of the GStreamer Pipeline
func (p *Pipeline) PushVideo(buffer []byte) {
	b := C.CBytes(buffer)
	defer C.free(b)
	C.gstreamer_receive_push_video_buffer(p.Pipeline, b, C.int(len(buffer)))
}

func (p *Pipeline) PushAudio(buffer []byte) {
	b := C.CBytes(buffer)
	defer C.free(b)
	C.gstreamer_receive_push_audio_buffer(p.Pipeline, b, C.int(len(buffer)))
}