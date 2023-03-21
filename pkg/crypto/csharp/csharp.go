// sources:
//   - https://referencesource.microsoft.com/#mscorlib/system/random.cs
//   - https://github.com/HirbodBehnam/CSharpRandom/blob/master/random.go
//   - https://github.com/MoonlightPS/Iridium-gidra/blob/master/gover/utils/prng.go
package csharp

import (
	"math"
	"math/rand"
)

const (
	MBIG  = math.MaxInt32
	MSEED = 161803398
)

type source struct {
	inext     int32
	inextp    int32
	seedArray [56]int32
}

func NewRand() *rand.Rand        { return rand.New(NewSource64()) }
func NewRand64() *rand.Rand      { return rand.New(NewSource64()) }
func NewSource() rand.Source     { return &source{} }
func NewSource64() rand.Source64 { return &source{} }

func (s *source) Seed(seed int64) { s.seed(int32(seed)) }
func (s *source) Int63() int64    { return int64(s.Uint64() & 0x7FFFFFFFFFFFFFFE) }
func (s *source) Uint64() uint64  { return uint64(s.sample() * math.MaxUint64) }

func (s *source) seed(seed int32) {
	var ii, mj, mk, subtraction int32
	// Initialize our Seed array.
	// This algorithm comes from Numerical Recipes in C (2nd Ed.)
	if seed == math.MinInt32 {
		subtraction = math.MaxInt32
	} else {
		subtraction = seed
		if subtraction < 0 {
			subtraction = -subtraction
		}
	}
	mj = MSEED - subtraction
	s.seedArray[55] = mj
	mk = 1
	// Apparently the range [1..55] is special (Knuth) and so we're wasting the 0'th position.
	for i := int32(1); i < 55; i++ {
		ii = (21 * i) % 55
		s.seedArray[ii] = mk
		mk = mj - mk
		if mk < 0 {
			mk += MBIG
		}
		mj = s.seedArray[ii]
	}
	for k := 1; k < 5; k++ {
		for i := 1; i < 56; i++ {
			s.seedArray[i] -= s.seedArray[1+(i+30)%55]
			if s.seedArray[i] < 0 {
				s.seedArray[i] += MBIG
			}
		}
	}
	s.inext = 0
	s.inextp = 21
}

func (s *source) sample() float64 {
	// Including this division at the end gives us significantly improved
	// random number distribution.
	return float64(s.internalSample()) * (1.0 / float64(MBIG))
}

func (s *source) internalSample() int32 {
	var retVal int32
	locINext := s.inext
	locINextp := s.inextp
	locINext++
	if locINext >= 56 {
		locINext = 1
	}
	locINextp++
	if locINextp >= 56 {
		locINextp = 1
	}
	retVal = s.seedArray[locINext] - s.seedArray[locINextp]
	if retVal == MBIG {
		retVal--
	}
	if retVal < 0 {
		retVal += MBIG
	}
	s.seedArray[locINext] = retVal
	s.inext = locINext
	s.inextp = locINextp
	return retVal
}
