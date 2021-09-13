package rist

import (
	/*
		#cgo LDFLAGS: -lrist
		#include "rist.h"
	*/
	"C"
)

type PeerConfig C.struct_rist_peer_config

func (p *PeerConfig) GetVersion() int  {
	return int(p.version);
}

func (p *PeerConfig) SetVersion(version int) {
	p.version = C.int(version);
}

func (p *PeerConfig) GetAddressFamily() int {
	return int(p.address_family);
}

func (p *PeerConfig) SetAddressFamily(addressFamily int) {
	p.address_family = C.int(addressFamily);
}

func (p *PeerConfig) GetInitiateConn() int {
	return int(p.initiate_conn);
}

func (p *PeerConfig) SetInitiateConn(initiateConn int) {
	p.initiate_conn = C.int(initiateConn);
}

func (p *PeerConfig) GetAddress() string {
	return C.GoString(&p.address[0]);
}

func (p *PeerConfig) SetAddress(address string) {
	arr := [256]C.char{}
	for i := 0; i < len(address) && i < 255; i++ {
		arr[i] = C.char(address[i])
	}
	p.address = arr
}

func (p *PeerConfig) GetMiface() string {
	return C.GoString(&p.miface[0]);
}

func (p *PeerConfig) SetMiface(miface string) {
	arr := [128]C.char{}
	for i := 0; i < len(miface) && i < 127; i++ {
		arr[i] = C.char(miface[i])
	}
	p.miface = arr
}

func (p *PeerConfig) GetPhysicalPort() uint16 {
	return uint16(p.physical_port);
}

func (p *PeerConfig) SetPhysicalPort(physicalPort uint16) {
	p.physical_port = C.uint16_t(physicalPort);
}

func (p *PeerConfig) GetVirtDstPort() uint16 {
	return uint16(p.virt_dst_port);
}

func (p *PeerConfig) SetVirtDstPort(virtDstPort uint16) {
	p.virt_dst_port = C.uint16_t(virtDstPort);
}

func (p *PeerConfig) GetRecoveryMode() uint32 {
	return p.recovery_mode;
}

func (p *PeerConfig) SetRecoveryMode(recoveryMode uint32) {
	p.recovery_mode = recoveryMode;
}

func (p *PeerConfig) GetRecoveryMaxbitrate() uint32 {
	return uint32(p.recovery_maxbitrate);
}

func (p *PeerConfig) SetRecoveryMaxbitrate(recoveryMaxbitrate uint32) {
	p.recovery_maxbitrate = C.uint32_t(recoveryMaxbitrate);
}

func (p *PeerConfig) GetRecoveryMaxbitrateReturn() uint32 {
	return uint32(p.recovery_maxbitrate_return);
}

func (p *PeerConfig) SetRecoveryMaxbitrateReturn(recoveryMaxbitrateReturn uint32) {
	p.recovery_maxbitrate_return = C.uint32_t(recoveryMaxbitrateReturn);
}

func (p *PeerConfig) GetRecoveryLengthMin() uint32 {
	return uint32(p.recovery_length_min);
}

func (p *PeerConfig) SetRecoveryLengthMin(recoveryLengthMin uint32) {
	p.recovery_length_min = C.uint32_t(recoveryLengthMin);
}

func (p *PeerConfig) GetRecoveryLengthMax() uint32 {
	return uint32(p.recovery_length_max);
}

func (p *PeerConfig) SetRecoveryLengthMax(recoveryLengthMax uint32) {
	p.recovery_length_max = C.uint32_t(recoveryLengthMax);
}

func (p *PeerConfig) GetRecoveryReorderBuffer() uint32 {
	return uint32(p.recovery_reorder_buffer);
}

func (p *PeerConfig) SetRecoveryReorderBuffer(recoveryReorderBuffer uint32) {
	p.recovery_reorder_buffer = C.uint32_t(recoveryReorderBuffer);
}

func (p *PeerConfig) GetRecoveryRttMin() uint32 {
	return uint32(p.recovery_rtt_min);
}

func (p *PeerConfig) SetRecoveryRttMin(recoveryRttMin uint32) {
	p.recovery_rtt_min = C.uint32_t(recoveryRttMin);
}

func (p *PeerConfig) GetRecoveryRttMax() uint32 {
	return uint32(p.recovery_rtt_max);
}

func (p *PeerConfig) SetRecoveryRttMax(recoveryRttMax uint32) {
	p.recovery_rtt_max = C.uint32_t(recoveryRttMax);
}

func (p *PeerConfig) GetWeight() uint32 {
	return uint32(p.weight);
}

func (p *PeerConfig) SetWeight(weight uint32) {
	p.weight = C.uint32_t(weight);
}

func (p *PeerConfig) GetSecret() string {
	return C.GoString(&p.secret[0]);
}

func (p *PeerConfig) SetSecret(secret string) {
	arr := [128]C.char{}
	for i := 0; i < len(secret) && i < 31; i++ {
		arr[i] = C.char(secret[i])
	}
	p.secret = arr
}

func (p *PeerConfig) GetKeySize() int {
	return int(p.key_size);
}

func (p *PeerConfig) SetKeySize(keySize int) {
	p.key_size = C.int(keySize);
}

func (p *PeerConfig) GetKeyRotation() uint32 {
	return uint32(p.key_rotation);
}

func (p *PeerConfig) SetKeyRotation(keyRotation uint32) {
	p.key_rotation = C.uint32_t(keyRotation);
}

func (p *PeerConfig) GetCompression() int {
	return int(p.compression);
}

func (p *PeerConfig) SetCompression(compression int) {
	p.compression = C.int(compression);
}

func (p *PeerConfig) GetCname() string {
	return C.GoString(&p.cname[0]);
}

func (p *PeerConfig) SetCname(cname string) {
	arr := [128]C.char{}
	for i := 0; i < len(cname) && i < 31; i++ {
		arr[i] = C.char(cname[i])
	}
	p.cname = arr
}

func (p *PeerConfig) GetCongestionControlMode() uint32 {
	return p.congestion_control_mode;
}

func (p *PeerConfig) SetCongestionControlMode(congestionControlMode uint32) {
	p.congestion_control_mode = congestionControlMode;
}

func (p *PeerConfig) GetMinRetries() uint32 {
	return uint32(p.min_retries);
}

func (p *PeerConfig) SetMinRetries(minRetries uint32) {
	p.min_retries = C.uint32_t(minRetries);
}

func (p *PeerConfig) GetMaxRetries() uint32 {
	return uint32(p.max_retries);
}

func (p *PeerConfig) SetMaxRetries(maxRetries uint32) {
	p.max_retries = C.uint32_t(maxRetries);
}

func (p *PeerConfig) GetSessionTimeout() uint32 {
	return uint32(p.session_timeout);
}

func (p *PeerConfig) SetSessionTimeout(sessionTimeout uint32) {
	p.session_timeout = C.uint32_t(sessionTimeout);
}

func (p *PeerConfig) GetKeepaliveInterval() uint32 {
	return uint32(p.keepalive_interval);
}

func (p *PeerConfig) SetKeepaliveInterval(keepaliveInterval uint32) {
	p.keepalive_interval = C.uint32_t(keepaliveInterval);
}

func (p *PeerConfig) GetTimingMode() uint32 {
	return uint32(p.timing_mode);
}

func (p *PeerConfig) SetTimingMode(timingMode uint32) {
	p.timing_mode = C.uint32_t(timingMode);
}

func (p *PeerConfig) GetSrpUsername() string {
	return C.GoString(&p.srp_username[0]);
}

func (p *PeerConfig) SetSrpUsername(username string) {
	arr := [256]C.char{}
	for i := 0; i < len(username) && i < 255; i++ {
		arr[i] = C.char(username[i])
	}
	p.srp_username = arr
}

func (p *PeerConfig) GetSrpPassword() string {
	return C.GoString(&p.srp_password[0]);
}

func (p *PeerConfig) SetSrpPassword(password string) {
	arr := [256]C.char{}
	for i := 0; i < len(password) && i < 255; i++ {
		arr[i] = C.char(password[i])
	}
	p.srp_password = arr
}

func (p *PeerConfig) Close() {
	var p2 *C.struct_rist_peer_config = (*C.struct_rist_peer_config)(p)
	C.rist_peer_config_free2(&p2)
}