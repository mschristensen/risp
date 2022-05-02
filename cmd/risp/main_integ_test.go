// build +integration
package main_test

import (
	"context"
	"risp/internal/app/apps"
	"risp/internal/app/cfg"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServerApp(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip()
	}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		s, err := apps.NewServerApp(cfg.PortFromEnv())
		require.NoError(t, err)
		require.NoError(t, s.Run(ctx, nil))
	}()
	go func() {
		defer wg.Done()
		c, err := apps.NewClientApp(cfg.PortFromEnv())
		require.NoError(t, err)
		require.NoError(t, c.Run(ctx, []string{"10"}))
		cancel()
	}()
	wg.Wait()
}
