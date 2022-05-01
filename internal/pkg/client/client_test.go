package client

import (
	"context"
	"encoding/binary"
	"testing"

	risppb "risp/api/proto/gen/pb-go/github.com/mschristensen/risp/api/build/go"
	"risp/api/proto/gen/pb-go/github.com/mschristensen/risp/api/build/go/mocks"
	"risp/internal/pkg/checksum"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var sequence []uint32 = []uint32{100, 50, 25, 15}

func TestRun(t *testing.T) {
	c, err := NewClient(
		WithSequenceLength(4),
	)
	require.NoError(t, err)
	ctx := context.Background()
	mockChannel := &mocks.RISP_ConnectClient{}
	c.channel = mockChannel
	mockChannel.On("Send", mock.IsType(&risppb.ClientMessage{})).Return(nil)
	for i, val := range sequence {
		offset := make([]byte, 2)
		binary.BigEndian.PutUint16(offset, uint16(i))
		mockChannel.On("Recv").Return(&risppb.ServerMessage{
			State:   risppb.ConnectionState_CONNECTED,
			Offset:  offset,
			Payload: val,
		}, nil).Once()
	}
	mockChannel.On("Recv").Return(&risppb.ServerMessage{
		State:    risppb.ConnectionState_FINALISING,
		Checksum: checksum.Sum(sequence...),
	}, nil).Once()
	mockChannel.On("Recv").Return(&risppb.ServerMessage{
		State: risppb.ConnectionState_CLOSING,
	}, nil).Once()
	mockChannel.On("Recv").Return(nil, nil).Maybe()
	require.NoError(t, c.Run(ctx))
	require.Equal(t, sequence, c.sequence)
}
