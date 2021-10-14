package internal

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0
#include "gst.h"
*/
import "C"
import (
	"log"
	"time"
	"unsafe"

	"github.com/mattn/go-pointer"
	"github.com/pion/webrtc/v3/pkg/media"
)

func init() {
	go C.gstreamer_main_loop()
	C.gstreamer_init()
}

type Pipeline struct {
	gstElement *C.GstElement

	RTCPSink       func([]byte) (int, error)
	VP8SampleSink  func(media.Sample) (int, error)
	OpusSampleSink func(media.Sample) (int, error)
}

func NewPipeline(RTCPSink func([]byte) (int, error), VP8SampleSink func(media.Sample) (int, error), OpusSampleSink func(media.Sample) (int, error)) *Pipeline {
	pipelineStr := C.CString(`
		rtpsession name=rtpsession rtp-profile=avpf sdes="application/x-rtp-source-sdes,cname=(string)\"mtun.io\""
			appsrc name=rtpappsrc is-live=true format=time ! rtpsession.recv_rtp_sink
			rtpsession.recv_rtp_src ! rtpjitterbuffer do-retransmission=true ! rtpptdemux name=demux
				demux.src_96 ! queue ! h265parse config-interval=-1 ! nvh265dec ! videoconvert ! vp8enc error-resilient=partitions keyframe-max-dist=10 auto-alt-ref=true cpu-used=5 deadline=1 ! queue ! appsink name=vp8appsink
				demux.src_111 ! queue ! opusparse ! queue ! appsink name=opusappsink
			rtpsession.send_rtcp_src ! appsink name=rtcpappsink async=false sync=false`)
	defer C.free(unsafe.Pointer(pipelineStr))

	p := &Pipeline{
		RTCPSink:       RTCPSink,
		VP8SampleSink:  VP8SampleSink,
		OpusSampleSink: OpusSampleSink,
	}
	p.gstElement = C.gstreamer_start(pipelineStr, pointer.Save(p))
	return p
}

// WriteRTP sends the given RTP packet to the pipeline for processing.
func (p *Pipeline) WriteRTP(buf []byte) error {
	b := C.CBytes(buf)
	defer C.free(b)

	codec := C.CString("rtpappsrc")
	defer C.free(unsafe.Pointer(codec))

	C.gstreamer_push_rtp(p.gstElement, codec, b, C.int(len(buf)))

	return nil
}

// Close closes the pipeline.
func (p *Pipeline) Close() {
	C.gstreamer_stop(p.gstElement)
}

var vp8Duration = 33333333
var opusDuration = 20000000

//export goHandleVP8Buffer
func goHandleVP8Buffer(buffer unsafe.Pointer, bufferLen C.int, timestamp C.ulong, data unsafe.Pointer) {
	if _, err := pointer.Restore(data).(*Pipeline).VP8SampleSink(media.Sample{Data: C.GoBytes(buffer, bufferLen), Duration: time.Duration(vp8Duration)}); err != nil {
		log.Printf("failed to write rtp packet: %v", err)
	}
}

//export goHandleOpusBuffer
func goHandleOpusBuffer(buffer unsafe.Pointer, bufferLen C.int, timestamp C.ulong, data unsafe.Pointer) {
	if _, err := pointer.Restore(data).(*Pipeline).OpusSampleSink(media.Sample{Data: C.GoBytes(buffer, bufferLen), Duration: time.Duration(opusDuration)}); err != nil {
		log.Printf("failed to write rtp packet: %v", err)
	}
}

//export goHandleRtcpAppSinkBuffer
func goHandleRtcpAppSinkBuffer(buffer unsafe.Pointer, bufferLen C.int, data unsafe.Pointer) {
	if _, err := pointer.Restore(data).(*Pipeline).RTCPSink(C.GoBytes(buffer, bufferLen)); err != nil {
		log.Printf("failed to write rtcp packet: %v", err)
	}
}
