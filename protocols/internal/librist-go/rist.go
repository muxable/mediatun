package rist

import (
	/*
		#cgo LDFLAGS: -lrist
		#include "rist.h"
	*/
	"C"
	"unsafe"

	gopointer "github.com/mattn/go-pointer"
)
import "time"

type RIST struct {
	ctx *C.struct_rist_ctx
	handlers []unsafe.Pointer
}

type Profile uint32

const (
	PROFILE_SIMPLE   = Profile(C.RIST_PROFILE_SIMPLE)
	PROFILE_MAIN     = Profile(C.RIST_PROFILE_MAIN)
	PROFILE_ADVANCED = Profile(C.RIST_PROFILE_ADVANCED)
)

type LoggingSettings C.struct_rist_logging_settings

type Peer C.struct_rist_peer

type AuthHandler interface {
	OnConnect(connecting_ip string, connecting_port uint16, local_ip string, local_port uint16, peer *Peer) int
	OnDisconnect(peer *Peer) int
}

//export goAuthHandlerOnConnect
func goAuthHandlerOnConnect(arg unsafe.Pointer, connecting_ip string, connecting_port uint16, local_ip string, local_port uint16, peer *C.struct_rist_peer) int {
	v := gopointer.Restore(arg).(AuthHandler)
	return v.OnConnect(connecting_ip, connecting_port, local_ip, local_port, (*Peer)(peer))
}

//export goAuthHandlerOnDisconnect
func goAuthHandlerOnDisconnect(arg unsafe.Pointer, peer *C.struct_rist_peer) int {
	v := gopointer.Restore(arg).(AuthHandler)
	return v.OnDisconnect((*Peer)(peer))
}

type ConnectionStatus C.enum_rist_connection_status

const (
	CONNECTION_ESTABLISHED = ConnectionStatus(C.RIST_CONNECTION_ESTABLISHED)
	CONNECTION_TIMED_OUT = ConnectionStatus(C.RIST_CONNECTION_TIMED_OUT)
	CLIENT_CONNECTED = ConnectionStatus(C.RIST_CLIENT_CONNECTED)
	CLIENT_TIMED_OUT = ConnectionStatus(C.RIST_CLIENT_TIMED_OUT)
)

type ConnectionStatusHandler interface {
	OnConnectionStatus(peer *Peer, peer_connection_status ConnectionStatus)
}

//export goConnectionStatusHandlerOnConnectionStatus
func goConnectionStatusHandlerOnConnectionStatus(arg unsafe.Pointer, peer *C.struct_rist_peer, peer_connection_status C.enum_rist_connection_status) {
	v := gopointer.Restore(arg).(ConnectionStatusHandler)
	v.OnConnectionStatus((*Peer)(peer), ConnectionStatus(peer_connection_status))
}

type OobBlock C.struct_rist_oob_block

func (b *OobBlock) GetPayload() []byte {
	return C.GoBytes(b.payload, C.int(uint64(b.payload_len)))
}

type OobHandler interface {
	OnReceiveOob(oob_block *OobBlock) int
}

//export goOobHandlerOnReceiveOob
func goOobHandlerOnReceiveOob(arg unsafe.Pointer, oob_block *C.struct_rist_oob_block) int {
	v := gopointer.Restore(arg).(OobHandler)
	return v.OnReceiveOob((*OobBlock)(oob_block))
}

type StatsHandler interface {
	OnReceiveStats(stats_container *Stats) int
}

//export goStatsHandlerOnReceiveStats
func goStatsHandlerOnReceiveStats(arg unsafe.Pointer, stats_container *C.struct_rist_stats) int {
	v := gopointer.Restore(arg).(StatsHandler)
	defer C.rist_stats_free(stats_container)
	return v.OnReceiveStats((*Stats)(stats_container))
}

type ReceiverDataHandler interface {
	OnData(data_block *DataBlock) int
}

//export goReceiverDataHandlerOnData
func goReceiverDataHandlerOnData(arg unsafe.Pointer, data_block *C.struct_rist_data_block) int {
	v := gopointer.Restore(arg).(ReceiverDataHandler)
	defer C.rist_receiver_data_block_free2(&data_block)
	return v.OnData((*DataBlock)(data_block))
}

func NewSender(profile Profile, flowId uint32, loggingSettings *LoggingSettings) *RIST {
	r := &RIST{}
	if (C.rist_sender_create(&r.ctx, uint32(profile), C.uint(flowId), (*C.struct_rist_logging_settings)(loggingSettings)) != 0) {
		return nil
	}
	return r
}

func NewReceiver(profile Profile, loggingSettings *LoggingSettings) *RIST {
	r := &RIST{}
	if (C.rist_receiver_create(&r.ctx, uint32(profile), (*C.struct_rist_logging_settings)(loggingSettings)) != 0) {
		return nil
	}
	return r
}

func (r *RIST) SetAuthHandler(handler AuthHandler) bool {
	p := gopointer.Save(handler)
	r.handlers = append(r.handlers, p)
	return C.rist_auth_handler_set(r.ctx, (*[0]byte)(C.cb_auth_connect), (*[0]byte)(C.cb_auth_disconnect), p) == 0
}

func (r *RIST) SetConnectionStatusHandler(handler ConnectionStatusHandler) bool {
	p := gopointer.Save(handler)
	r.handlers = append(r.handlers, p)
	return C.rist_connection_status_callback_set(r.ctx, (C.connection_status_callback_t)(C.cb_connection_status), p) == 0
}

func (r *RIST) SetOobHandler(handler OobHandler) bool {
	p := gopointer.Save(handler)
	r.handlers = append(r.handlers, p)
	return C.rist_oob_callback_set(r.ctx, (*[0]byte)(C.cb_recv_oob), p) == 0
}

func (r *RIST) SetStatsHandler(interval time.Duration, handler StatsHandler) bool {
	p := gopointer.Save(handler)
	r.handlers = append(r.handlers, p)
	return C.rist_stats_callback_set(r.ctx, C.int(interval.Milliseconds()), (*[0]byte)(C.cb_stats), p) == 0
}

func (r *RIST) SetReceiverDataHandler(handler ReceiverDataHandler) bool {
	p := gopointer.Save(handler)
	r.handlers = append(r.handlers, p)
	return C.rist_receiver_data_callback_set2(r.ctx, (C.receiver_data_callback2_t)(C.cb_recv), p) == 0
}

func (r *RIST) SenderDataWrite(dataBlock *DataBlock) bool {
	return C.rist_sender_data_write(r.ctx, (*C.struct_rist_data_block)(dataBlock)) == 0

}

func ParseAddress(input string) *PeerConfig {
	s := C.CString(input)
	defer C.free(unsafe.Pointer(s))
    var c *C.struct_rist_peer_config
	if C.rist_parse_address2(s, &c) != 0 {
		return nil
	}
	return (*PeerConfig)(c)
}

func (r *RIST) PeerCreate(config *PeerConfig) *Peer {
	var p *C.struct_rist_peer
	if C.rist_peer_create(r.ctx, &p, (*C.struct_rist_peer_config)(config)) != 0 {
		return nil
	}
	return (*Peer)(p)
}

func (r *RIST) PeerDestroy(peer *Peer) bool {
	return C.rist_peer_destroy(r.ctx, (*C.struct_rist_peer)(peer)) == 0
}

func (r *RIST) Start() bool {
	return C.rist_start(r.ctx) == 0
}

func (r *RIST) Close() {
	C.rist_destroy(r.ctx)
	for _, handler := range r.handlers {
		gopointer.Unref(handler)
	}
}