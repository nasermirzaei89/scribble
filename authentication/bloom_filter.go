package authentication

import (
	"hash/fnv"
	"math"
	"sync"
)

type BloomFilter struct {
	mu        sync.RWMutex
	bits      []bool
	numBits   uint
	numHashes uint
}

func NewBloomFilter(expectedItems uint, falsePositiveRate float64) *BloomFilter {
	if expectedItems == 0 {
		expectedItems = 1
	}

	m := optimalBitCount(expectedItems, falsePositiveRate)
	k := optimalHashCount(m, expectedItems)

	return &BloomFilter{
		bits:      make([]bool, m),
		numBits:   m,
		numHashes: k,
	}
}

func optimalBitCount(n uint, p float64) uint {
	m := -float64(n) * math.Log(p) / (math.Log(2) * math.Log(2))
	return uint(math.Ceil(m))
}

func optimalHashCount(m, n uint) uint {
	k := uint(math.Round(float64(m) / float64(n) * math.Log(2)))
	if k < 1 {
		return 1
	}

	return k
}

func (bf *BloomFilter) hashPositions(item string) []uint {
	// h1: FNV-1a 32-bit
	h1 := fnv.New32a()
	h1.Write([]byte(item))
	v1 := uint(h1.Sum32())

	// h2: FNV-1 32-bit (note: different variant, always odd to ensure full coverage)
	h2 := fnv.New32()
	h2.Write([]byte(item))
	v2 := uint(h2.Sum32()) | 1

	positions := make([]uint, bf.numHashes)
	for i := uint(0); i < bf.numHashes; i++ {
		positions[i] = (v1 + i*v2) % bf.numBits
	}

	return positions
}

func (bf *BloomFilter) Add(item string) {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	for _, pos := range bf.hashPositions(item) {
		bf.bits[pos] = true
	}
}

// Returns false → item is DEFINITELY NOT in the set.
// Returns true  → item MIGHT be in the set (a false positive is possible).
func (bf *BloomFilter) Test(item string) bool {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	for _, pos := range bf.hashPositions(item) {
		if !bf.bits[pos] {
			return false
		}
	}

	return true
}
