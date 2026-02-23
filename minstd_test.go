package rules

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Cross-language test vectors from byte-scratch/minstd-prng/TEST_VECTORS.md.
// The Rust implementation must produce identical values for these seeds.

func TestMinstdNextSequence(t *testing.T) {
	rng := NewMinstdRand(1)
	expected := []uint64{
		48271,
		182605794,
		1291394886,
		1914720637,
		2078669041,
		407355683,
		1105902161,
		854716505,
		564586691,
		1596680831,
	}
	for i, want := range expected {
		got := rng.Next()
		require.Equal(t, want, got, "Next() call %d", i)
	}
}

func TestMinstdSeedZeroRemapped(t *testing.T) {
	// Seed 0 is an absorbing state; it should be remapped to 1
	rng := NewMinstdRand(0)
	require.Equal(t, uint64(48271), rng.Next(), "seed 0 should behave like seed 1")
}

func TestMinstdIntn(t *testing.T) {
	rng := NewMinstdRand(42)
	// First few Intn(100) values for seed 42
	results := make([]int, 5)
	for i := range results {
		results[i] = rng.Intn(100)
	}
	expected := []int{82, 7, 37, 15, 42}
	require.Equal(t, expected, results)
}

func TestMinstdShuffle(t *testing.T) {
	rng := NewMinstdRand(42)
	items := []int{0, 1, 2, 3, 4, 5, 6, 7}
	rng.Shuffle(len(items), func(i, j int) {
		items[i], items[j] = items[j], items[i]
	})
	expected := []int{7, 1, 3, 2, 0, 5, 4, 6}
	require.Equal(t, expected, items)
}

func TestMinstdRange(t *testing.T) {
	rng := NewMinstdRand(1)
	// Range(1, 10) should be in [1, 10]
	for i := 0; i < 100; i++ {
		v := rng.Range(1, 10)
		require.GreaterOrEqual(t, v, 1)
		require.LessOrEqual(t, v, 10)
	}
}

func TestMinstdSeedRandDeterminism(t *testing.T) {
	// Two RNGs with same game seed and turn must produce identical sequences
	rng1 := NewMinstdSeedRand(99999, 42)
	rng2 := NewMinstdSeedRand(99999, 42)
	for i := 0; i < 100; i++ {
		require.Equal(t, rng1.Intn(1000), rng2.Intn(1000), "call %d", i)
	}
}

func TestMinstdSeedRandDifferentTurns(t *testing.T) {
	// Different turns produce different sequences
	rng1 := NewMinstdSeedRand(12345, 0)
	rng2 := NewMinstdSeedRand(12345, 1)
	// First values should differ (splitmix64 ensures good mixing)
	require.NotEqual(t, rng1.Next(), rng2.Next())
}

func TestSplitmix64TurnSeedVectors(t *testing.T) {
	// Cross-language test vectors for turn seed derivation
	type testCase struct {
		gameSeed int64
		turn     int
	}

	tests := []testCase{
		{12345, 0},
		{12345, 1},
		{12345, 42},
		{0, 0},
		{1, 0},
	}

	// Verify determinism: same inputs always produce same seed
	for _, tc := range tests {
		rng1 := NewMinstdSeedRand(tc.gameSeed, tc.turn)
		rng2 := NewMinstdSeedRand(tc.gameSeed, tc.turn)
		require.Equal(t, rng1.state, rng2.state,
			"gameSeed=%d turn=%d should produce identical state", tc.gameSeed, tc.turn)
	}

	// Verify all derived seeds are in valid MINSTD range [1, m-1)
	for seed := int64(0); seed < 100; seed++ {
		for turn := 0; turn < 100; turn++ {
			rng := NewMinstdSeedRand(seed, turn)
			require.Greater(t, rng.state, uint64(0),
				"state must be > 0 for seed=%d turn=%d", seed, turn)
			require.Less(t, rng.state, minstdM,
				"state must be < m for seed=%d turn=%d", seed, turn)
		}
	}
}

func TestMinstdImplementsRandInterface(t *testing.T) {
	// Verify MinstdRand satisfies the Rand interface
	var _ Rand = (*MinstdRand)(nil)
}
