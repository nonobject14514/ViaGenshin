package transport

import (
	"sync"
)

var packetPool = &sync.Pool{
	New: func() interface{} { return make(Payload, 256*1200) },
}

type Payload []byte

func NewPayload(n int) Payload {
	return packetPool.New().(Payload)[:n]
}

func (p *Payload) Release() {
	packetPool.Put(*p)
}
