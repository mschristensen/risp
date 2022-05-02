package client

import (
	"context"
	"fmt"
	"testing"

	risppb "risp/api/proto/gen/pb-go/github.com/mschristensen/risp/api/build/go"
	"risp/api/proto/gen/pb-go/github.com/mschristensen/risp/api/build/go/mocks"
	"risp/internal/pkg/session"
	"risp/pkg/checksum"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var seqs []session.Sequence = []session.Sequence{
	session.Uint32SliceToSequence([]uint32{}),
	session.Uint32SliceToSequence([]uint32{1, 2}),
	session.Uint32SliceToSequence([]uint32{0, 0, 0, 0}),
	session.Uint32SliceToSequence([]uint32{100, 50, 25, 15, 10}),
}

func TestRun(t *testing.T) {
	t.Parallel()
	for i := range seqs {
		j := i
		t.Run(fmt.Sprintf("test_%d", j), func(t *testing.T) {
			t.Parallel()
			c, err := NewClient(
				WithSequenceLength(uint16(len(seqs[j]))),
			)
			require.NoError(t, err)
			ctx := context.Background()
			mockChannel := &mocks.RISP_ConnectClient{}
			c.channel = mockChannel
			// TODO should assert expected sequence of messages from client
			mockChannel.On("Send", mock.IsType(&risppb.ClientMessage{})).Return(nil)
			for idx, val := range seqs[j] {
				mockChannel.On("Recv").Return(&risppb.ServerMessage{
					State:   risppb.ConnectionState_CONNECTED,
					Index:   uint32(idx),
					Payload: *val,
				}, nil).Once()
			}
			sum, err := checksum.Sum(seqs[j]...)
			require.NoError(t, err)
			mockChannel.On("Recv").Return(&risppb.ServerMessage{
				State:    risppb.ConnectionState_CLOSING,
				Checksum: sum,
			}, nil).Once()
			mockChannel.On("Recv").Return(&risppb.ServerMessage{
				State: risppb.ConnectionState_CLOSED,
			}, nil).Once()
			mockChannel.On("Recv").Return(nil, nil).Maybe()
			require.NoError(t, c.Run(ctx))
			require.Equal(t, seqs[j].ToUint32Slice(), c.session.Sequence.ToUint32Slice())
		})
	}
}
