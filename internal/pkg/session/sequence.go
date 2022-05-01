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
