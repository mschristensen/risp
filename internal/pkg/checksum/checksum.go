package checksum

import "encoding/binary"

// Sum adds up all the uint32 values and returns the result as a byte array.
// TODO: use binary arithmetic algorithm to avoid overflow.
func Sum(elems ...uint32) []byte {
	var sum uint32
	for _, elem := range elems {
		sum += elem
	}
	result := make([]byte, 8)
	binary.BigEndian.PutUint32(result, sum)
	return result
}
