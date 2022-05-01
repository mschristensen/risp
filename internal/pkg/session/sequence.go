package session

import (
	"fmt"
	"strings"
)

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
