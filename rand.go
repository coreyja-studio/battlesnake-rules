package rules

import "math/rand"

type Rand interface {
	Intn(n int) int
	// Range produces a random integer in the range of [min,max] (inclusive)
	// For example, Range(1,3) could produce the values 1, 2 or 3.
	// Panics if max < min (like how Intn(n) panics for n <=0)
	Range(min, max int) int
	Shuffle(n int, swap func(i, j int))
}

// A Rand implementation that just uses the global math/rand generator.
var GlobalRand globalRand

type globalRand struct{}

func (globalRand) Range(min, max int) int {
	return rand.Intn(max-min+1) + min
}

func (globalRand) Intn(n int) int {
	return rand.Intn(n)
}

func (globalRand) Shuffle(n int, swap func(i, j int)) {
	rand.Shuffle(n, swap)
}

type seedRand struct {
	seed int64
	rand *rand.Rand
}

func NewSeedRand(seed int64) *seedRand {
	return &seedRand{
		seed: seed,
		rand: rand.New(rand.NewSource(seed)),
	}
}

func (s seedRand) Intn(n int) int {
	return s.rand.Intn(n)
}

func (s seedRand) Range(min, max int) int {
	return s.rand.Intn(max-min+1) + min
}

func (s seedRand) Shuffle(n int, swap func(i, j int)) {
	s.rand.Shuffle(n, swap)
}

// MINSTD Park-Miller PRNG (revised, 1993).
// Portable across Go and Rust for deterministic cross-engine verification.
// Algorithm: state(n+1) = (state(n) * 48271) mod 2147483647
const (
	minstdA          uint64 = 48271
	minstdM          uint64 = 2147483647 // 2^31 - 1
	splitmix64Golden uint64 = 0x9e3779b97f4a7c15
)

type MinstdRand struct {
	state uint64
}

func NewMinstdRand(seed int64) *MinstdRand {
	s := uint64(seed) % (minstdM - 1)
	if s == 0 {
		s = 1
	}
	return &MinstdRand{state: s}
}

func (r *MinstdRand) Next() uint64 {
	r.state = (r.state * minstdA) % minstdM
	return r.state
}

func (r *MinstdRand) Intn(n int) int {
	return int(r.Next() % uint64(n))
}

func (r *MinstdRand) Range(min, max int) int {
	return r.Intn(max-min+1) + min
}

func (r *MinstdRand) Shuffle(n int, swap func(i, j int)) {
	for i := n - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		swap(i, j)
	}
}

// splitmix64Mix is a finalizer for decorrelating adjacent seeds.
func splitmix64Mix(x uint64) uint64 {
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	x ^= x >> 31
	return x
}

// NewMinstdSeedRand creates a MINSTD PRNG seeded for a specific turn.
// Uses splitmix64 to mix gameSeed and turn, preventing correlated sequences.
func NewMinstdSeedRand(gameSeed int64, turn int) *MinstdRand {
	combined := (uint64(gameSeed) ^ uint64(turn)) + splitmix64Golden
	raw := splitmix64Mix(combined)
	bounded := (raw % (minstdM - 1)) + 1
	return NewMinstdRand(int64(bounded))
}

// For testing purposes

// A Rand implementation that always returns the minimum value for any method.
var MinRand minRand

type minRand struct{}

func (minRand) Intn(n int) int {
	return 0
}

func (minRand) Range(min, max int) int {
	return min
}

func (minRand) Shuffle(n int, swap func(i, j int)) {
	// no shuffling
}

// A Rand implementation that always returns the maximum value for any method.
var MaxRand maxRand

type maxRand struct{}

func (maxRand) Intn(n int) int {
	return n - 1
}

func (maxRand) Range(min, max int) int {
	return max
}

func (maxRand) Shuffle(n int, swap func(i, j int)) {
	// rotate by one element so every element is moved
	if n < 2 {
		return
	}
	for i := 0; i < n-2; i++ {
		swap(i, i+1)
	}
	swap(n-2, n-1)
}
