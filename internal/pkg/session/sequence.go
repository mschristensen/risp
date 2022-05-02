package session

import (
	"fmt"
	"strings"
)

// Sequence is the sequence of numbers to transmit using the RISP protocol.
type Sequence []*uint32

func (s Sequence) String() string {
	arr := make([]string, len(s))
	for i := range s {
		if s[i] == nil {
			continue
		}
		arr[i] = fmt.Sprintf("%d", *s[i])
	}
	return strings.Join(arr, " ")
}

// ToUint32Slice converts the sequence to a slice of uint32.
// If an element is nil, the zero value of uint32 is used.
func (s Sequence) ToUint32Slice() []uint32 {
	arr := make([]uint32, len(s))
	for i := range s {
		if s[i] == nil {
			continue
		}
		arr[i] = *s[i]
	}
	return arr
}

// Uint32SliceToSequence converts a slice of uint32 to a sequence.
func Uint32SliceToSequence(arr []uint32) Sequence {
	s := make(Sequence, len(arr))
	for i := range arr {
		x := arr[i]
		s[i] = &x
	}
	return s
}
