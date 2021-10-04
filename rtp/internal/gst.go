package internal

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-app-1.0
#include "gst.h"
*/
import "C"
import (
	"log"
	"unsafe"

	"github.com/mattn/go-pointer"
)

func init() {
	go C.gstreamer_main_loop()
	C.gstreamer_init()
}

type Pipeline struct {
	gstElement *C.GstElement
	
	RTCPSink func([]byte) (int, error)
	RTPSink  func([]byte) (int, error)
}

type PipelineType int

const (
	PipelineTypeVideo = PipelineType(0)
	PipelineTypeAudio = PipelineType(1)
)

func (p *Pipeline) Start(pipelineType PipelineType) error {
	switch pipelineType {
	case PipelineTypeVideo:
		pipelineStr := C.CString(`
			rtpsession name=rtpsession rtp-profile=avpf sdes="application/x-rtp-source-sdes,cname=(string)\"mtun.io\""
				appsrc name=rtpappsrc is-live=true format=time caps="application/x-rtp,media=(string)video,clock-rate=(int)90000,encoding-name=(string)VP8-DRAFT-IETF-01,payload=(int)120,extmap-5=http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01" !
					rtpsession.recv_rtp_sink
				rtpsession.recv_rtp_src !
					rtprtxreceive payload-type-map="application/x-rtp-pt-map,120=(uint)121" !
					rtpjitterbuffer do-lost=true do-retransmission=true name=rtpjitterbuffer ! appsink name=bufferappsink
				rtpsession.send_rtcp_src ! appsink name=rtcpappsink sync=false async=false`)
		defer C.free(unsafe.Pointer(pipelineStr))

		p.gstElement = C.gstreamer_start(pipelineStr, pointer.Save(p))
	case PipelineTypeAudio:
		pipelineStr := C.CString(`
			rtpsession name=rtpsession rtp-profile=avpf sdes="application/x-rtp-source-sdes,cname=(string)\"mtun.io\""
				appsrc name=rtpappsrc is-live=true format=time caps="application/x-rtp,media=(string)audio,clock-rate=(int)48000,encoding-name=(string)OPUS,payload=(int)96" !
					rtpsession.recv_rtp_sink
				rtpsession.recv_rtp_src !
					rtprtxreceive payload-type-map="application/x-rtp-pt-map,96=(uint)97" !
					rtpjitterbuffer do-lost=true do-retransmission=true name=rtpjitterbuffer ! appsink name=bufferappsink
				rtpsession.send_rtcp_src ! appsink name=rtcpappsink sync=false async=false`)
		defer C.free(unsafe.Pointer(pipelineStr))

		p.gstElement = C.gstreamer_start(pipelineStr, pointer.Save(p))

	}
	return nil
}

// WriteRTP sends the given RTP packet to the pipeline for processing.
func (p *Pipeline) WriteRTP(buffer []byte) error {
	b := C.CBytes(buffer)
	defer C.free(b)
	C.gstreamer_push_rtp(p.gstElement, b, C.int(len(buffer)))

	return nil
}

// WriteRTCP sends the given RTCP packet to the pipeline for processing.
func (p *Pipeline) WriteRTCP(buffer []byte) error {
	b := C.CBytes(buffer)
	defer C.free(b)
	C.gstreamer_push_rtcp(p.gstElement, b, C.int(len(buffer)))

	return nil
}

// Close terminates the pipeline.
func (p *Pipeline) Close() {
	C.gstreamer_stop(p.gstElement)
}

//export goHandleAppSinkBuffer
func goHandleAppSinkBuffer(buffer unsafe.Pointer, bufferLen C.int, duration C.int, data unsafe.Pointer) {
	defer C.free(buffer)
	pipeline := pointer.Restore(data).(*Pipeline)
	if pipeline.RTPSink != nil {
		if _, err := pipeline.RTPSink(C.GoBytes(buffer, bufferLen)); err != nil {
			log.Printf("failed to write rtp packet: %v", err)
		}
	} else {
		panic("missing buffer sink")
	}
}

//export goHandleRtcpAppSinkBuffer
func goHandleRtcpAppSinkBuffer(buffer unsafe.Pointer, bufferLen C.int, duration C.int, data unsafe.Pointer) {
	pipeline := pointer.Restore(data).(*Pipeline)
	if pipeline.RTCPSink != nil {
		if _, err := pipeline.RTCPSink(C.GoBytes(buffer, bufferLen)); err != nil {
			log.Printf("failed to write rtcp packet: %v", err)
		}
	} else {
		panic("missing buffer sink")
	}
}
