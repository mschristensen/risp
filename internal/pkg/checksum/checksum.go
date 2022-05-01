package checksum

import (
	"errors"
	"math"
)

// ErrSequenceTooLong is returned when the sequence contains too many values.
var ErrSequenceTooLong = errors.New("sequence too long")

// ErrUnexpectedNilElement is returned when the sequence unexpectedly contains a nil value.
var ErrUnexpectedNilElement = errors.New("unexpected nil value in sequence")

// Sum adds up all the uint32 values in the sequence and returns the result.
// If the sequence contains nil values, an error is returned.
// The maximum number of values in the sequence is 2^16-1 (0xffff).
// Therefore, the largest sum that can be calculated is (2^16-1) * (2^32-1) = 2^48-2^32-2^16+1.
// Log2(2^48-2^32-2^16+1) = 48, so we use a 64-bit integer to store the sum.
func Sum(sequence ...*uint32) (uint64, error) {
	if len(sequence) > math.MaxUint16 {
		return 0, ErrSequenceTooLong
	}
	var sum uint64
	for _, elem := range sequence {
		if elem == nil {
			return 0, ErrUnexpectedNilElement
		}
		sum += uint64(*elem)
	}
	return sum, nil
}
