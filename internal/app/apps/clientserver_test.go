package apps

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServerApp(t *testing.T) {
	s, err := NewServerApp()
	require.NoError(t, err)
	require.NoError(t, s.Run(context.Background(), nil))
}
