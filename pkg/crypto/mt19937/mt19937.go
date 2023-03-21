package mt19937

import (
	"encoding/binary"
	"math/rand"
)

const (
	N          = 312
	M          = 156
	MATRIX_A   = 0xB5026F5AA96619E9
	UPPER_MASK = 0xFFFFFFFF80000000
	LOWER_MASK = 0x7FFFFFFF
)

type source struct {
	array [312]uint64 // state vector
	index uint64      // array index
}

func NewRand() *rand.Rand        { return rand.New(NewSource64()) }
func NewRand64() *rand.Rand      { return rand.New(NewSource64()) }
func NewSource() rand.Source     { return &source{index: N + 1} }
func NewSource64() rand.Source64 { return &source{index: N + 1} }

func (s *source) Seed(seed int64) {
	s.array[0] = uint64(seed)
	for s.index = 1; s.index < N; s.index++ {
		s.array[s.index] = 0x5851F42D4C957F2D*(s.array[s.index-1]^(s.array[s.index-1]>>62)) + s.index
	}
}

func (s *source) Int63() int64 {
	return int64(s.Uint64() & 0x7FFFFFFFFFFFFFFE)
}

func (s *source) Uint64() uint64 {
	var i int
	var x uint64
	magic := []uint64{0, MATRIX_A}
	if s.index >= N {
		if s.index == N+1 {
			s.Seed(int64(5489))
		}
		for i = 0; i < N-M; i++ {
			x = (s.array[i] & UPPER_MASK) | (s.array[i+1] & LOWER_MASK)
			s.array[i] = s.array[i+(M)] ^ (x >> 1) ^ magic[int(x&uint64(1))]
		}
		for ; i < N-1; i++ {
			x = (s.array[i] & UPPER_MASK) | (s.array[i+1] & LOWER_MASK)
			s.array[i] = s.array[i+(M-N)] ^ (x >> 1) ^ magic[int(x&uint64(1))]
		}
		x = (s.array[N-1] & UPPER_MASK) | (s.array[0] & LOWER_MASK)
		s.array[N-1] = s.array[M-1] ^ (x >> 1) ^ magic[int(x&uint64(1))]
		s.index = 0
	}
	x = s.array[s.index]
	s.index++
	x ^= (x >> 29) & 0x5555555555555555
	x ^= (x << 17) & 0x71D67FFFEDA60000
	x ^= (x << 37) & 0xFFF7EEE000000000
	x ^= x >> 43
	return x
}

type KeyBlock struct {
	seed uint64
	data [4096]byte
}

func NewKeyBlock(seed uint64) *KeyBlock {
	b := &KeyBlock{seed: seed}
	r := NewRand()
	r.Seed(int64(b.seed))
	r.Seed(int64(r.Uint64()))
	r.Uint64()
	for i := 0; i < 4096>>3; i++ {
		binary.BigEndian.PutUint64(b.data[i<<3:], r.Uint64())
	}
	return b
}

func (b *KeyBlock) Key() []byte {
	return b.data[:]
}

func (b *KeyBlock) Seed() uint64 {
	return b.seed
}

func (b *KeyBlock) Xor(data []byte) {
	for i := 0; i < len(data); i++ {
		data[i] ^= b.data[i%4096]
	}
}
