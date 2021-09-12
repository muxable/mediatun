package gst

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0
#include "gst.h"
*/
import "C"
import (
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/pion/webrtc/v3/pkg/media"
)

func init() {
	go C.gstreamer_start_mainloop()
}

// Pipeline is a wrapper for a GStreamer Pipeline
type Pipeline struct {
	Pipeline  *C.GstElement
	onSample  func(media.Sample)
	id        int
	codecName string
}

var pipelines = make(map[int]*Pipeline)
var pipelinesLock sync.Mutex

// CreatePipeline creates a GStreamer Pipeline
func CreatePipeline(pipelineStr string) *Pipeline {
	pipelineStrUnsafe := C.CString(pipelineStr)
	defer C.free(unsafe.Pointer(pipelineStrUnsafe))

	pipelinesLock.Lock()
	defer pipelinesLock.Unlock()

	pipeline := &Pipeline{
		Pipeline:  C.gstreamer_create_pipeline(pipelineStrUnsafe),
		id:        len(pipelines),
	}

	pipelines[pipeline.id] = pipeline
	return pipeline
}

// Set the sample callback.
func (p *Pipeline) SetOnSample(onSample func(media.Sample)) {
	p.onSample = onSample
}

// Start starts the GStreamer Pipeline
func (p *Pipeline) Start() {
	if p.onSample == nil {
		panic("OnSample callback not set")
	}
	C.gstreamer_start_pipeline(p.Pipeline, C.int(p.id))
}

// Stop stops the GStreamer Pipeline
func (p *Pipeline) Stop() {
	C.gstreamer_stop_pipeline(p.Pipeline)
}

//export goHandlePipelineBuffer
func goHandlePipelineBuffer(buffer unsafe.Pointer, bufferLen C.int, duration C.int, pipelineID C.int) {
	pipelinesLock.Lock()
	pipeline, ok := pipelines[int(pipelineID)]
	pipelinesLock.Unlock()

	if ok {
		pipeline.onSample(media.Sample{Data: C.GoBytes(buffer, bufferLen), Duration: time.Duration(duration)})
	} else {
		fmt.Printf("discarding buffer, no pipeline with id %d", int(pipelineID))
	}
}
