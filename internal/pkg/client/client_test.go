package client

import (
	"context"
	"testing"

	risppb "risp/api/proto/gen/pb-go/github.com/mschristensen/risp/api/build/go"
	"risp/api/proto/gen/pb-go/github.com/mschristensen/risp/api/build/go/mocks"
	"risp/pkg/checksum"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func uint32Ptr(x uint32) *uint32 {
	return &x
}

var sequence []*uint32 = []*uint32{
	uint32Ptr(100),
	uint32Ptr(50),
	uint32Ptr(25),
	uint32Ptr(15),
}

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
		mockChannel.On("Recv").Return(&risppb.ServerMessage{
			State:   risppb.ConnectionState_CONNECTED,
			Index:   uint32(i),
			Payload: *val,
		}, nil).Once()
	}
	sum, err := checksum.Sum(sequence...)
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
	require.Equal(t, sequence, c.session.Sequence)
}
