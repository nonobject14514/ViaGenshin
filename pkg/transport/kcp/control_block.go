package kcp

import (
	"encoding/binary"
	"sync"
	"time"
)

const (
	IKCP_RTO_NDL     = 30  // no delay min rto
	IKCP_RTO_MIN     = 100 // normal min rto
	IKCP_RTO_DEF     = 200
	IKCP_RTO_MAX     = 60000
	IKCP_CMD_PUSH    = 81 // cmd: push data
	IKCP_CMD_ACK     = 82 // cmd: ack
	IKCP_CMD_WASK    = 83 // cmd: window probe (ask)
	IKCP_CMD_WINS    = 84 // cmd: window size (tell)
	IKCP_ASK_SEND    = 1  // need to send IKCP_CMD_WASK
	IKCP_ASK_TELL    = 2  // need to send IKCP_CMD_WINS
	IKCP_WND_SND     = 32
	IKCP_WND_RCV     = 32
	IKCP_MTU_DEF     = 1400
	IKCP_ACK_FAST    = 3
	IKCP_INTERVAL    = 100
	IKCP_OVERHEAD    = 28
	IKCP_DEADLINK    = 20
	IKCP_THRESH_INIT = 2
	IKCP_THRESH_MIN  = 2
	IKCP_PROBE_INIT  = 7000   // 7 secs to probe window size
	IKCP_PROBE_LIMIT = 120000 // up to 120 secs to probe window
	IKCP_SN_OFFSET   = 16
)

type ControlBlock struct {
	convID    uint32
	sessionID uint32

	output OutputFunc

	mtu, mss, state                        uint32
	snd_una, snd_nxt, rcv_nxt              uint32
	ssthresh                               uint32
	rx_rttvar, rx_srtt                     int32
	rx_rto, rx_minrto                      uint32
	snd_wnd, rcv_wnd, rmt_wnd, cwnd, probe uint32
	interval, ts_flush                     uint32
	nodelay, updated                       uint32
	ts_probe, probe_wait                   uint32
	dead_link, incr                        uint32

	fastresend     int32
	nocwnd, stream int32

	snd_queue []segmentData
	rcv_queue []segmentData
	snd_buf   []segmentData
	rcv_buf   []segmentData

	acklist []ackItem

	buffer   []byte
	reserved int
}

type OutputFunc func([]byte)

func NewControlBlock(convID, sessionID uint32, output OutputFunc) *ControlBlock {
	cb := new(ControlBlock)
	cb.convID = convID
	cb.sessionID = sessionID
	cb.output = output

	cb.snd_wnd = IKCP_WND_SND
	cb.rcv_wnd = IKCP_WND_RCV
	cb.rmt_wnd = IKCP_WND_RCV
	cb.mtu = IKCP_MTU_DEF
	cb.mss = cb.mtu - IKCP_OVERHEAD
	cb.buffer = make([]byte, cb.mtu)
	cb.rx_rto = IKCP_RTO_DEF
	cb.rx_minrto = IKCP_RTO_MIN
	cb.interval = IKCP_INTERVAL
	cb.ts_flush = IKCP_INTERVAL
	cb.ssthresh = IKCP_THRESH_INIT
	cb.dead_link = IKCP_DEADLINK
	return cb
}

func (cb *ControlBlock) Input(data []byte, regular, ackNoDelay bool) int {
	snd_una := cb.snd_una
	if len(data) < IKCP_OVERHEAD {
		return -1
	}

	var latest uint32 // the latest ack packet
	var flag int
	var inSegs uint64
	var windowSlides bool

	for {
		var ts, sn, length, una, conv, token uint32
		var wnd uint16
		var cmd, frg uint8

		if len(data) < int(IKCP_OVERHEAD) {
			break
		}

		data = ikcp_decode32u(data, &conv)
		if conv != cb.convID {
			return -1
		}

		data = ikcp_decode32u(data, &token)
		if token != cb.sessionID {
			return -4
		}

		data = ikcp_decode8u(data, &cmd)
		data = ikcp_decode8u(data, &frg)
		data = ikcp_decode16u(data, &wnd)
		data = ikcp_decode32u(data, &ts)
		data = ikcp_decode32u(data, &sn)
		data = ikcp_decode32u(data, &una)
		data = ikcp_decode32u(data, &length)
		if len(data) < int(length) {
			return -2
		}

		if cmd != IKCP_CMD_PUSH && cmd != IKCP_CMD_ACK &&
			cmd != IKCP_CMD_WASK && cmd != IKCP_CMD_WINS {
			return -3
		}

		// only trust window updates from regular packets. i.e: latest update
		if regular {
			cb.rmt_wnd = uint32(wnd)
		}
		if cb.parse_una(una) > 0 {
			windowSlides = true
		}
		cb.shrink_buf()

		if cmd == IKCP_CMD_ACK {
			cb.parse_ack(sn)
			cb.parse_fastack(sn, ts)
			flag |= 1
			latest = ts
		} else if cmd == IKCP_CMD_PUSH {
			// repeat := true
			if _itimediff(sn, cb.rcv_nxt+cb.rcv_wnd) < 0 {
				cb.ack_push(sn, ts)
				if _itimediff(sn, cb.rcv_nxt) >= 0 {
					var seg segmentData
					seg.convID = conv
					seg.sessionID = token
					seg.cmd = cmd
					seg.frg = frg
					seg.wnd = wnd
					seg.ts = ts
					seg.sn = sn
					seg.una = una
					seg.body = data[:length] // delayed data copying
					// repeat = cb.parse_data(seg)
					_ = cb.parse_data(seg)
				}
			}
			// if regular && repeat {
			// 	atomic.AddUint64(&DefaultSnmp.RepeatSegs, 1)
			// }
		} else if cmd == IKCP_CMD_WASK {
			// ready to send back IKCP_CMD_WINS in Ikcp_flush
			// tell remote my window size
			cb.probe |= IKCP_ASK_TELL
		} else if cmd == IKCP_CMD_WINS {
			// do nothing
		} else {
			return -3
		}

		inSegs++
		data = data[length:]
	}
	// atomic.AddUint64(&DefaultSnmp.InSegs, inSegs)

	// update rtt with the latest ts
	// ignore the FEC packet
	if flag != 0 && regular {
		current := currentMs()
		if _itimediff(current, latest) >= 0 {
			cb.update_ack(_itimediff(current, latest))
		}
	}

	// cwnd update when packet arrived
	if cb.nocwnd == 0 {
		if _itimediff(cb.snd_una, snd_una) > 0 {
			if cb.cwnd < cb.rmt_wnd {
				mss := cb.mss
				if cb.cwnd < cb.ssthresh {
					cb.cwnd++
					cb.incr += mss
				} else {
					if cb.incr < mss {
						cb.incr = mss
					}
					cb.incr += (mss*mss)/cb.incr + (mss / 16)
					if (cb.cwnd+1)*mss <= cb.incr {
						if mss > 0 {
							cb.cwnd = (cb.incr + mss - 1) / mss
						} else {
							cb.cwnd = cb.incr + mss - 1
						}
					}
				}
				if cb.cwnd > cb.rmt_wnd {
					cb.cwnd = cb.rmt_wnd
					cb.incr = cb.rmt_wnd * mss
				}
			}
		}
	}

	if windowSlides { // if window has slided, flush
		cb.flush(false)
	} else if ackNoDelay && len(cb.acklist) > 0 { // ack immediately
		cb.flush(true)
	}
	return 0
}

const (
	// 1500 is the default MTU of most networks
	MAX_MTU = 1500
)

var (
	// a system-wide packet buffer shared among sending, receiving and FEC
	// to mitigate high-frequency memory allocation for packets, bytes from xmitBuf
	// is aligned to 64bit
	xmitBuf sync.Pool
)

func init() {
	xmitBuf.New = func() interface{} {
		return make([]byte, MAX_MTU)
	}
}

// monotonic reference time point
var refTime time.Time = time.Now()

// currentMs returns current elapsed monotonic milliseconds since program startup
func currentMs() uint32 { return uint32(time.Since(refTime) / time.Millisecond) }

/* encode 8 bits unsigned int */
func ikcp_encode8u(p []byte, c byte) []byte {
	p[0] = c
	return p[1:]
}

/* decode 8 bits unsigned int */
func ikcp_decode8u(p []byte, c *byte) []byte {
	*c = p[0]
	return p[1:]
}

/* encode 16 bits unsigned int (lsb) */
func ikcp_encode16u(p []byte, w uint16) []byte {
	binary.LittleEndian.PutUint16(p, w)
	return p[2:]
}

/* decode 16 bits unsigned int (lsb) */
func ikcp_decode16u(p []byte, w *uint16) []byte {
	*w = binary.LittleEndian.Uint16(p)
	return p[2:]
}

/* encode 32 bits unsigned int (lsb) */
func ikcp_encode32u(p []byte, l uint32) []byte {
	binary.LittleEndian.PutUint32(p, l)
	return p[4:]
}

/* decode 32 bits unsigned int (lsb) */
func ikcp_decode32u(p []byte, l *uint32) []byte {
	*l = binary.LittleEndian.Uint32(p)
	return p[4:]
}

func _imin_(a, b uint32) uint32 {
	if a <= b {
		return a
	}
	return b
}

func _imax_(a, b uint32) uint32 {
	if a >= b {
		return a
	}
	return b
}

func _ibound_(lower, middle, upper uint32) uint32 {
	return _imin_(_imax_(lower, middle), upper)
}

func _itimediff(later, earlier uint32) int32 {
	return (int32)(later - earlier)
}

type ackItem struct {
	sn uint32
	ts uint32
}

// newSegment creates a KCP segmentData
func (cb *ControlBlock) newSegment(size int) (seg segmentData) {
	seg.body = xmitBuf.Get().([]byte)[:size]
	return
}

// delSegment recycles a KCP segmentData
func (cb *ControlBlock) delSegment(seg *segmentData) {
	if seg.body != nil {
		xmitBuf.Put(seg.body)
		seg.body = nil
	}
}

// ReserveBytes keeps n bytes untouched from the beginning of the buffer,
// the output_callback function should be aware of this.
//
// Return false if n >= mss
func (cb *ControlBlock) ReserveBytes(n int) bool {
	if n >= int(cb.mtu-IKCP_OVERHEAD) || n < 0 {
		return false
	}
	cb.reserved = n
	cb.mss = cb.mtu - IKCP_OVERHEAD - uint32(n)
	return true
}

// PeekSize checks the size of next message in the recv queue
func (cb *ControlBlock) PeekSize() (length int) {
	if len(cb.rcv_queue) == 0 {
		return -1
	}

	seg := &cb.rcv_queue[0]
	if seg.frg == 0 {
		return len(seg.body)
	}

	if len(cb.rcv_queue) < int(seg.frg+1) {
		return -1
	}

	for k := range cb.rcv_queue {
		seg := &cb.rcv_queue[k]
		length += len(seg.body)
		if seg.frg == 0 {
			break
		}
	}
	return
}

// Receive data from kcp state machine
//
// Return number of bytes read.
//
// Return -1 when there is no readable data.
//
// Return -2 if len(buffer) is smaller than kcp.PeekSize().
func (cb *ControlBlock) Recv(buffer []byte) (n int) {
	peeksize := cb.PeekSize()
	if peeksize < 0 {
		return -1
	}

	if peeksize > len(buffer) {
		return -2
	}

	var fast_recover bool
	if len(cb.rcv_queue) >= int(cb.rcv_wnd) {
		fast_recover = true
	}

	// merge fragment
	count := 0
	for k := range cb.rcv_queue {
		seg := &cb.rcv_queue[k]
		copy(buffer, seg.body)
		buffer = buffer[len(seg.body):]
		n += len(seg.body)
		count++
		cb.delSegment(seg)
		if seg.frg == 0 {
			break
		}
	}
	if count > 0 {
		cb.rcv_queue = cb.remove_front(cb.rcv_queue, count)
	}

	// move available data from rcv_buf -> rcv_queue
	count = 0
	for k := range cb.rcv_buf {
		seg := &cb.rcv_buf[k]
		if seg.sn == cb.rcv_nxt && len(cb.rcv_queue)+count < int(cb.rcv_wnd) {
			cb.rcv_nxt++
			count++
		} else {
			break
		}
	}

	if count > 0 {
		cb.rcv_queue = append(cb.rcv_queue, cb.rcv_buf[:count]...)
		cb.rcv_buf = cb.remove_front(cb.rcv_buf, count)
	}

	// fast recover
	if len(cb.rcv_queue) < int(cb.rcv_wnd) && fast_recover {
		// ready to send back IKCP_CMD_WINS in ikcp_flush
		// tell remote my window size
		cb.probe |= IKCP_ASK_TELL
	}
	return
}

// Send is user/upper level send, returns below zero for error
func (cb *ControlBlock) Send(buffer []byte) int {
	var count int
	if len(buffer) == 0 {
		return -1
	}

	// append to previous segmentData in streaming mode (if possible)
	if cb.stream != 0 {
		n := len(cb.snd_queue)
		if n > 0 {
			seg := &cb.snd_queue[n-1]
			if len(seg.body) < int(cb.mss) {
				capacity := int(cb.mss) - len(seg.body)
				extend := capacity
				if len(buffer) < capacity {
					extend = len(buffer)
				}

				// grow slice, the underlying cap is guaranteed to
				// be larger than cb.mss
				oldlen := len(seg.body)
				seg.body = seg.body[:oldlen+extend]
				copy(seg.body[oldlen:], buffer)
				buffer = buffer[extend:]
			}
		}

		if len(buffer) == 0 {
			return 0
		}
	}

	if len(buffer) <= int(cb.mss) {
		count = 1
	} else {
		count = (len(buffer) + int(cb.mss) - 1) / int(cb.mss)
	}

	if count > 255 {
		return -2
	}

	if count == 0 {
		count = 1
	}

	for i := 0; i < count; i++ {
		var size int
		if len(buffer) > int(cb.mss) {
			size = int(cb.mss)
		} else {
			size = len(buffer)
		}
		seg := cb.newSegment(size)
		copy(seg.body, buffer[:size])
		if cb.stream == 0 { // message mode
			seg.frg = uint8(count - i - 1)
		} else { // stream mode
			seg.frg = 0
		}
		cb.snd_queue = append(cb.snd_queue, seg)
		buffer = buffer[size:]
	}
	return 0
}

func (cb *ControlBlock) update_ack(rtt int32) {
	// https://tools.ietf.org/html/rfc6298
	var rto uint32
	if cb.rx_srtt == 0 {
		cb.rx_srtt = rtt
		cb.rx_rttvar = rtt >> 1
	} else {
		delta := rtt - cb.rx_srtt
		cb.rx_srtt += delta >> 3
		if delta < 0 {
			delta = -delta
		}
		if rtt < cb.rx_srtt-cb.rx_rttvar {
			// if the new RTT sample is below the bottom of the range of
			// what an RTT measurement is expected to be.
			// give an 8x reduced weight versus its normal weighting
			cb.rx_rttvar += (delta - cb.rx_rttvar) >> 5
		} else {
			cb.rx_rttvar += (delta - cb.rx_rttvar) >> 2
		}
	}
	rto = uint32(cb.rx_srtt) + _imax_(cb.interval, uint32(cb.rx_rttvar)<<2)
	cb.rx_rto = _ibound_(cb.rx_minrto, rto, IKCP_RTO_MAX)
}

func (cb *ControlBlock) shrink_buf() {
	if len(cb.snd_buf) > 0 {
		seg := &cb.snd_buf[0]
		cb.snd_una = seg.sn
	} else {
		cb.snd_una = cb.snd_nxt
	}
}

func (cb *ControlBlock) parse_ack(sn uint32) {
	if _itimediff(sn, cb.snd_una) < 0 || _itimediff(sn, cb.snd_nxt) >= 0 {
		return
	}

	for k := range cb.snd_buf {
		seg := &cb.snd_buf[k]
		if sn == seg.sn {
			// mark and free space, but leave the segmentData here,
			// and wait until `una` to delete this, then we don't
			// have to shift the segments behind forward,
			// which is an expensive operation for large window
			seg.acked = 1
			cb.delSegment(seg)
			break
		}
		if _itimediff(sn, seg.sn) < 0 {
			break
		}
	}
}

func (cb *ControlBlock) parse_fastack(sn, ts uint32) {
	if _itimediff(sn, cb.snd_una) < 0 || _itimediff(sn, cb.snd_nxt) >= 0 {
		return
	}

	for k := range cb.snd_buf {
		seg := &cb.snd_buf[k]
		if _itimediff(sn, seg.sn) < 0 {
			break
		} else if sn != seg.sn && _itimediff(seg.ts, ts) <= 0 {
			seg.fastack++
		}
	}
}

func (cb *ControlBlock) parse_una(una uint32) int {
	count := 0
	for k := range cb.snd_buf {
		seg := &cb.snd_buf[k]
		if _itimediff(una, seg.sn) > 0 {
			cb.delSegment(seg)
			count++
		} else {
			break
		}
	}
	if count > 0 {
		cb.snd_buf = cb.remove_front(cb.snd_buf, count)
	}
	return count
}

// ack append
func (cb *ControlBlock) ack_push(sn, ts uint32) {
	cb.acklist = append(cb.acklist, ackItem{sn, ts})
}

// returns true if data has repeated
func (cb *ControlBlock) parse_data(newseg segmentData) bool {
	sn := newseg.sn
	if _itimediff(sn, cb.rcv_nxt+cb.rcv_wnd) >= 0 ||
		_itimediff(sn, cb.rcv_nxt) < 0 {
		return true
	}

	n := len(cb.rcv_buf) - 1
	insert_idx := 0
	repeat := false
	for i := n; i >= 0; i-- {
		seg := &cb.rcv_buf[i]
		if seg.sn == sn {
			repeat = true
			break
		}
		if _itimediff(sn, seg.sn) > 0 {
			insert_idx = i + 1
			break
		}
	}

	if !repeat {
		// replicate the content if it's new
		dataCopy := xmitBuf.Get().([]byte)[:len(newseg.body)]
		copy(dataCopy, newseg.body)
		newseg.body = dataCopy

		if insert_idx == n+1 {
			cb.rcv_buf = append(cb.rcv_buf, newseg)
		} else {
			cb.rcv_buf = append(cb.rcv_buf, segmentData{})
			copy(cb.rcv_buf[insert_idx+1:], cb.rcv_buf[insert_idx:])
			cb.rcv_buf[insert_idx] = newseg
		}
	}

	// move available data from rcv_buf -> rcv_queue
	count := 0
	for k := range cb.rcv_buf {
		seg := &cb.rcv_buf[k]
		if seg.sn == cb.rcv_nxt && len(cb.rcv_queue)+count < int(cb.rcv_wnd) {
			cb.rcv_nxt++
			count++
		} else {
			break
		}
	}
	if count > 0 {
		cb.rcv_queue = append(cb.rcv_queue, cb.rcv_buf[:count]...)
		cb.rcv_buf = cb.remove_front(cb.rcv_buf, count)
	}

	return repeat
}

func (cb *ControlBlock) wnd_unused() uint16 {
	if len(cb.rcv_queue) < int(cb.rcv_wnd) {
		return uint16(int(cb.rcv_wnd) - len(cb.rcv_queue))
	}
	return 0
}

// flush pending data
func (cb *ControlBlock) flush(ackOnly bool) uint32 {
	var seg segmentData
	seg.convID = cb.convID
	seg.sessionID = cb.sessionID
	seg.cmd = IKCP_CMD_ACK
	seg.wnd = cb.wnd_unused()
	seg.una = cb.rcv_nxt

	buffer := cb.buffer
	ptr := buffer[cb.reserved:] // keep n bytes untouched

	// makeSpace makes room for writing
	makeSpace := func(space int) {
		size := len(buffer) - len(ptr)
		if size+space > int(cb.mtu) {
			cb.output(buffer[:size])
			ptr = buffer[cb.reserved:]
		}
	}

	// flush bytes in buffer if there is any
	flushBuffer := func() {
		size := len(buffer) - len(ptr)
		if size > cb.reserved {
			cb.output(buffer[:size])
		}
	}

	// flush acknowledges
	for i, ack := range cb.acklist {
		makeSpace(IKCP_OVERHEAD)
		// filter jitters caused by bufferbloat
		if _itimediff(ack.sn, cb.rcv_nxt) >= 0 || len(cb.acklist)-1 == i {
			seg.sn, seg.ts = ack.sn, ack.ts
			ptr = seg.encode(ptr)
		}
	}
	cb.acklist = cb.acklist[0:0]

	if ackOnly { // flash remain ack segments
		flushBuffer()
		return cb.interval
	}

	// probe window size (if remote window size equals zero)
	if cb.rmt_wnd == 0 {
		current := currentMs()
		if cb.probe_wait == 0 {
			cb.probe_wait = IKCP_PROBE_INIT
			cb.ts_probe = current + cb.probe_wait
		} else {
			if _itimediff(current, cb.ts_probe) >= 0 {
				if cb.probe_wait < IKCP_PROBE_INIT {
					cb.probe_wait = IKCP_PROBE_INIT
				}
				cb.probe_wait += cb.probe_wait / 2
				if cb.probe_wait > IKCP_PROBE_LIMIT {
					cb.probe_wait = IKCP_PROBE_LIMIT
				}
				cb.ts_probe = current + cb.probe_wait
				cb.probe |= IKCP_ASK_SEND
			}
		}
	} else {
		cb.ts_probe = 0
		cb.probe_wait = 0
	}

	// flush window probing commands
	if (cb.probe & IKCP_ASK_SEND) != 0 {
		seg.cmd = IKCP_CMD_WASK
		makeSpace(IKCP_OVERHEAD)
		ptr = seg.encode(ptr)
	}

	// flush window probing commands
	if (cb.probe & IKCP_ASK_TELL) != 0 {
		seg.cmd = IKCP_CMD_WINS
		makeSpace(IKCP_OVERHEAD)
		ptr = seg.encode(ptr)
	}

	cb.probe = 0

	// calculate window size
	cwnd := _imin_(cb.snd_wnd, cb.rmt_wnd)
	if cb.nocwnd == 0 {
		cwnd = _imin_(cb.cwnd, cwnd)
	}

	// sliding window, controlled by snd_nxt && sna_una+cwnd
	newSegsCount := 0
	for k := range cb.snd_queue {
		if _itimediff(cb.snd_nxt, cb.snd_una+cwnd) >= 0 {
			break
		}
		newseg := cb.snd_queue[k]
		newseg.convID = cb.convID
		newseg.sessionID = cb.sessionID
		newseg.cmd = IKCP_CMD_PUSH
		newseg.sn = cb.snd_nxt
		cb.snd_buf = append(cb.snd_buf, newseg)
		cb.snd_nxt++
		newSegsCount++
	}
	if newSegsCount > 0 {
		cb.snd_queue = cb.remove_front(cb.snd_queue, newSegsCount)
	}

	// calculate resent
	resent := uint32(cb.fastresend)
	if cb.fastresend <= 0 {
		resent = 0xffffffff
	}

	// check for retransmissions
	current := currentMs()
	var change, lostSegs, fastRetransSegs, earlyRetransSegs uint64
	minrto := int32(cb.interval)

	ref := cb.snd_buf[:len(cb.snd_buf)] // for bounds check elimination
	for k := range ref {
		segment := &ref[k]
		needsend := false
		if segment.acked == 1 {
			continue
		}
		if segment.xmit == 0 { // initial transmit
			needsend = true
			segment.rto = cb.rx_rto
			segment.resendts = current + segment.rto
		} else if segment.fastack >= resent { // fast retransmit
			needsend = true
			segment.fastack = 0
			segment.rto = cb.rx_rto
			segment.resendts = current + segment.rto
			change++
			fastRetransSegs++
		} else if segment.fastack > 0 && newSegsCount == 0 { // early retransmit
			needsend = true
			segment.fastack = 0
			segment.rto = cb.rx_rto
			segment.resendts = current + segment.rto
			change++
			earlyRetransSegs++
		} else if _itimediff(current, segment.resendts) >= 0 { // RTO
			needsend = true
			if cb.nodelay == 0 {
				segment.rto += cb.rx_rto
			} else {
				segment.rto += cb.rx_rto / 2
			}
			segment.fastack = 0
			segment.resendts = current + segment.rto
			lostSegs++
		}

		if needsend {
			current = currentMs()
			segment.xmit++
			segment.ts = current
			segment.wnd = seg.wnd
			segment.una = seg.una

			need := IKCP_OVERHEAD + len(segment.body)
			makeSpace(need)
			ptr = segment.encode(ptr)
			copy(ptr, segment.body)
			ptr = ptr[len(segment.body):]

			if segment.xmit >= cb.dead_link {
				cb.state = 0xFFFFFFFF
			}
		}

		// get the nearest rto
		if rto := _itimediff(segment.resendts, current); rto > 0 && rto < minrto {
			minrto = rto
		}
	}

	// flash remain segments
	flushBuffer()

	// counter updates
	// sum := lostSegs
	// if lostSegs > 0 {
	// 	atomic.AddUint64(&DefaultSnmp.LostSegs, lostSegs)
	// }
	// if fastRetransSegs > 0 {
	// 	atomic.AddUint64(&DefaultSnmp.FastRetransSegs, fastRetransSegs)
	// 	sum += fastRetransSegs
	// }
	// if earlyRetransSegs > 0 {
	// 	atomic.AddUint64(&DefaultSnmp.EarlyRetransSegs, earlyRetransSegs)
	// 	sum += earlyRetransSegs
	// }
	// if sum > 0 {
	// 	atomic.AddUint64(&DefaultSnmp.RetransSegs, sum)
	// }

	// cwnd update
	if cb.nocwnd == 0 {
		// update ssthresh
		// rate halving, https://tools.ietf.org/html/rfc6937
		if change > 0 {
			inflight := cb.snd_nxt - cb.snd_una
			cb.ssthresh = inflight / 2
			if cb.ssthresh < IKCP_THRESH_MIN {
				cb.ssthresh = IKCP_THRESH_MIN
			}
			cb.cwnd = cb.ssthresh + resent
			cb.incr = cb.cwnd * cb.mss
		}

		// congestion control, https://tools.ietf.org/html/rfc5681
		if lostSegs > 0 {
			cb.ssthresh = cwnd / 2
			if cb.ssthresh < IKCP_THRESH_MIN {
				cb.ssthresh = IKCP_THRESH_MIN
			}
			cb.cwnd = 1
			cb.incr = cb.mss
		}

		if cb.cwnd < 1 {
			cb.cwnd = 1
			cb.incr = cb.mss
		}
	}

	return uint32(minrto)
}

// (deprecated)
//
// Update updates state (call it repeatedly, every 10ms-100ms), or you can ask
// ikcp_check when to call it again (without ikcp_input/_send calling).
// 'current' - current timestamp in millisec.
func (cb *ControlBlock) Update() {
	var slap int32

	current := currentMs()
	if cb.updated == 0 {
		cb.updated = 1
		cb.ts_flush = current
	}

	slap = _itimediff(current, cb.ts_flush)

	if slap >= 10000 || slap < -10000 {
		cb.ts_flush = current
		slap = 0
	}

	if slap >= 0 {
		cb.ts_flush += cb.interval
		if _itimediff(current, cb.ts_flush) >= 0 {
			cb.ts_flush = current + cb.interval
		}
		cb.flush(false)
	}
}

// (deprecated)
//
// Check determines when should you invoke ikcp_update:
// returns when you should invoke ikcp_update in millisec, if there
// is no ikcp_input/_send calling. you can call ikcp_update in that
// time, instead of call update repeatly.
// Important to reduce unnacessary ikcp_update invoking. use it to
// schedule ikcp_update (eg. implementing an epoll-like mechanism,
// or optimize ikcp_update when handling massive kcp connections)
func (cb *ControlBlock) Check() uint32 {
	current := currentMs()
	ts_flush := cb.ts_flush
	tm_flush := int32(0x7fffffff)
	tm_packet := int32(0x7fffffff)
	minimal := uint32(0)
	if cb.updated == 0 {
		return current
	}

	if _itimediff(current, ts_flush) >= 10000 ||
		_itimediff(current, ts_flush) < -10000 {
		ts_flush = current
	}

	if _itimediff(current, ts_flush) >= 0 {
		return current
	}

	tm_flush = _itimediff(ts_flush, current)

	for k := range cb.snd_buf {
		seg := &cb.snd_buf[k]
		diff := _itimediff(seg.resendts, current)
		if diff <= 0 {
			return current
		}
		if diff < tm_packet {
			tm_packet = diff
		}
	}

	minimal = uint32(tm_packet)
	if tm_packet >= tm_flush {
		minimal = uint32(tm_flush)
	}
	if minimal >= cb.interval {
		minimal = cb.interval
	}

	return current + minimal
}

// SetMtu changes MTU size, default is 1400
func (cb *ControlBlock) SetMtu(mtu int) int {
	if mtu < 50 || mtu < IKCP_OVERHEAD {
		return -1
	}
	if cb.reserved >= int(cb.mtu-IKCP_OVERHEAD) || cb.reserved < 0 {
		return -1
	}

	buffer := make([]byte, mtu)
	if buffer == nil {
		return -2
	}
	cb.mtu = uint32(mtu)
	cb.mss = cb.mtu - IKCP_OVERHEAD - uint32(cb.reserved)
	cb.buffer = buffer
	return 0
}

// NoDelay options
// fastest: ikcp_nodelay(kcp, 1, 20, 2, 1)
// nodelay: 0:disable(default), 1:enable
// interval: internal update timer interval in millisec, default is 100ms
// resend: 0:disable fast resend(default), 1:enable fast resend
// nc: 0:normal congestion control(default), 1:disable congestion control
func (cb *ControlBlock) NoDelay(nodelay, interval, resend, nc int) int {
	if nodelay >= 0 {
		cb.nodelay = uint32(nodelay)
		if nodelay != 0 {
			cb.rx_minrto = IKCP_RTO_NDL
		} else {
			cb.rx_minrto = IKCP_RTO_MIN
		}
	}
	if interval >= 0 {
		if interval > 5000 {
			interval = 5000
		} else if interval < 10 {
			interval = 10
		}
		cb.interval = uint32(interval)
	}
	if resend >= 0 {
		cb.fastresend = int32(resend)
	}
	if nc >= 0 {
		cb.nocwnd = int32(nc)
	}
	return 0
}

// WndSize sets maximum window size: sndwnd=32, rcvwnd=32 by default
func (cb *ControlBlock) WndSize(sndwnd, rcvwnd int) int {
	if sndwnd > 0 {
		cb.snd_wnd = uint32(sndwnd)
	}
	if rcvwnd > 0 {
		cb.rcv_wnd = uint32(rcvwnd)
	}
	return 0
}

// WaitSnd gets how many packet is waiting to be sent
func (cb *ControlBlock) WaitSnd() int {
	return len(cb.snd_buf) + len(cb.snd_queue)
}

// remove front n elements from queue
// if the number of elements to remove is more than half of the size.
// just shift the rear elements to front, otherwise just reslice q to q[n:]
// then the cost of runtime.growslice can always be less than n/2
func (cb *ControlBlock) remove_front(q []segmentData, n int) []segmentData {
	if n > cap(q)/2 {
		newn := copy(q, q[n:])
		return q[:newn]
	}
	return q[n:]
}

// Release all cached outgoing segments
func (cb *ControlBlock) ReleaseTX() {
	for k := range cb.snd_queue {
		if cb.snd_queue[k].body != nil {
			xmitBuf.Put(cb.snd_queue[k].body)
		}
	}
	for k := range cb.snd_buf {
		if cb.snd_buf[k].body != nil {
			xmitBuf.Put(cb.snd_buf[k].body)
		}
	}
	cb.snd_queue = nil
	cb.snd_buf = nil
}
