package kcp

import (
	"encoding/hex"
	"net"
	"sync"
	"unsafe"
)

const (
	controlCommandSyn = 0x000000FF
	controlCommandAck = 0x00000145
	controlCommandFin = 0x00000194

	controlMessageClientAppID = 1234567890
	controlMessageEditorAppID = 987654321
)

type controlData [20]byte

func (b *controlData) Set(command, convID, sessionID, message uint32) {
	b[0], b[1], b[2], b[3] = byte(command>>24), byte(command>>16), byte(command>>8), byte(command)
	b[4], b[5], b[6], b[7] = byte(convID>>24), byte(convID>>16), byte(convID>>8), byte(convID)
	b[8], b[9], b[10], b[11] = byte(sessionID>>24), byte(sessionID>>16), byte(sessionID>>8), byte(sessionID)
	b[12], b[13], b[14], b[15] = byte(message>>24), byte(message>>16), byte(message>>8), byte(message)
	// fill the last 4 bytes with magic number
	if command == controlCommandSyn {
		b[16], b[17], b[18], b[19] = 0xFF, 0xFF, 0xFF, 0xFF
	} else if command == controlCommandAck {
		b[16], b[17], b[18], b[19] = 0x14, 0x51, 0x45, 0x45
	} else if command == controlCommandFin {
		b[16], b[17], b[18], b[19] = 0x19, 0x41, 0x94, 0x94
	}
}

func (b *controlData) Command() uint32 {
	// do not check the last 4 bytes with magic number
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

func (b *controlData) ConvID() uint32 {
	return uint32(b[4])<<24 | uint32(b[5])<<16 | uint32(b[6])<<8 | uint32(b[7])
}

func (b *controlData) SessionID() uint32 {
	return uint32(b[8])<<24 | uint32(b[9])<<16 | uint32(b[10])<<8 | uint32(b[11])
}

func (b *controlData) Message() uint32 {
	return uint32(b[12])<<24 | uint32(b[13])<<16 | uint32(b[14])<<8 | uint32(b[15])
}

var controlDataPool = sync.Pool{
	New: func() any { return new(controlData) },
}

type segmentHead [28]byte

func (b *segmentHead) Set(convID, sessionID uint32, cmd, frg uint8, wnd uint16, ts, sn, una uint32, length int) {
	b[0], b[1], b[2], b[3] = byte(convID), byte(convID>>8), byte(convID>>16), byte(convID>>24)
	b[4], b[5], b[6], b[7] = byte(sessionID), byte(sessionID>>8), byte(sessionID>>16), byte(sessionID>>24)
	b[8], b[9], b[10], b[11] = cmd, frg, byte(wnd), byte(wnd>>8)
	b[12], b[13], b[14], b[15] = byte(ts), byte(ts>>8), byte(ts>>16), byte(ts>>24)
	b[16], b[17], b[18], b[19] = byte(sn), byte(sn>>8), byte(sn>>16), byte(sn>>24)
	b[20], b[21], b[22], b[23] = byte(una), byte(una>>8), byte(una>>16), byte(una>>24)
	b[24], b[25], b[26], b[27] = byte(length), byte(length>>8), byte(length>>16), byte(length>>24)
}

func (b *segmentHead) ConvID() uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func (b *segmentHead) SessionID() uint32 {
	return uint32(b[4]) | uint32(b[5])<<8 | uint32(b[6])<<16 | uint32(b[7])<<24
}

func (b *segmentHead) Cmd() uint8 {
	return b[8]
}

func (b *segmentHead) Frg() uint8 {
	return b[9]
}

func (b *segmentHead) Wnd() uint16 {
	return uint16(b[10]) | uint16(b[11])<<8
}

func (b *segmentHead) Ts() uint32 {
	return uint32(b[12]) | uint32(b[13])<<8 | uint32(b[14])<<16 | uint32(b[15])<<24
}

func (b *segmentHead) Sn() uint32 {
	return uint32(b[16]) | uint32(b[17])<<8 | uint32(b[18])<<16 | uint32(b[19])<<24
}

func (b *segmentHead) Una() uint32 {
	return uint32(b[20]) | uint32(b[21])<<8 | uint32(b[22])<<16 | uint32(b[23])<<24
}

func (b *segmentHead) Length() int {
	return int(uint32(b[24]) | uint32(b[25])<<8 | uint32(b[26])<<16 | uint32(b[27])<<24)
}

func (b *segmentHead) New(body []byte) *segmentData {
	data := &segmentData{}
	data.convID = b.ConvID()
	data.sessionID = b.SessionID()
	data.cmd = b.Cmd()
	data.frg = b.Frg()
	data.wnd = b.Wnd()
	data.ts = b.Ts()
	data.sn = b.Sn()
	data.una = b.Una()
	data.body = segmentBodyPool.Get().([]byte)[:b.Length()]
	copy(data.body, body)
	return data
}

var segmentHeadPool = sync.Pool{
	New: func() any { return &segmentHead{} },
}

type segmentData struct {
	convID    uint32
	sessionID uint32
	cmd       uint8
	frg       uint8
	wnd       uint16
	ts        uint32
	sn        uint32
	una       uint32
	body      []byte

	rto      uint32
	xmit     uint32
	resendts uint32
	fastack  uint32
	acked    uint32 // mark if the seg has acked
}

func (seg *segmentData) encode(ptr []byte) []byte {
	ptr = ikcp_encode32u(ptr, seg.convID)
	ptr = ikcp_encode32u(ptr, seg.sessionID)
	ptr = ikcp_encode8u(ptr, seg.cmd)
	ptr = ikcp_encode8u(ptr, seg.frg)
	ptr = ikcp_encode16u(ptr, seg.wnd)
	ptr = ikcp_encode32u(ptr, seg.ts)
	ptr = ikcp_encode32u(ptr, seg.sn)
	ptr = ikcp_encode32u(ptr, seg.una)
	ptr = ikcp_encode32u(ptr, uint32(len(seg.body)))
	return ptr
}

var segmentBodyPool = sync.Pool{
	New: func() any { return make([]byte, DefaultMTU) },
}

type onControlDataFunc func(*controlData, *net.UDPAddr) error
type onSegmentDataFunc func([]byte, *net.UDPAddr) error

func loopReadFromUDP(conn *net.UDPConn, onControlData onControlDataFunc, onSegmentData onSegmentDataFunc) {
	b := make([]byte, DefaultMTU)
	for {
		n, addr, err := conn.ReadFromUDP(b)
		if err != nil {
			logging(LoggingLevelError, "read error: %v", err)
			return
		}
		// unsafe pointer to avoid copy
		if n == 20 {
			logging(LoggingLevelTrace, "received control data %d from %v\n%s", n, addr, hex.Dump(b[:n]))
			data := (*controlData)(unsafe.Pointer(&b[0]))
			err = onControlData(data, addr)
		} else if n >= 28 {
			err = onSegmentData(b[:n], addr)
		} else {
			err = ErrInvalidPacket
		}
		if err != nil {
			logging(LoggingLevelError, "receive error: %v from %v", err, addr)
		}
	}
}

func writeControlDataToUDP(conn *net.UDPConn, data *controlData, addr *net.UDPAddr) error {
	_, err := conn.WriteToUDP(data[:], addr)
	logging(LoggingLevelTrace, "sending control data %d to %v\n%s", len(data), addr, hex.Dump(data[:]))
	return err
}

func writeToUDP(conn *net.UDPConn, data []byte, addr *net.UDPAddr) error {
	_, err := conn.WriteToUDP(data, addr)
	return err
}
