package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNgrok_NgrokStop(t *testing.T) {
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())

	ng := Ngrok{
		NgrokCtx: []NgCtx{
			{
				VMendpoint: "endpoint1",
				CtxCancel:  cancel1,
				Ctx:        ctx1,
			},
			{
				VMendpoint: "endpoint2",
				CtxCancel:  cancel2,
				Ctx:        ctx2,
			},
		},
	}

	// Before cancel, contexts should not be done
	assert.False(t, isContextDone(ctx1))
	assert.False(t, isContextDone(ctx2))

	ng.NgrokStop("endpoint1")

	// After cancel, ctx1 should be done
	assert.True(t, isContextDone(ctx1))
	assert.False(t, isContextDone(ctx2))

	ng.NgrokStop("endpoint2")

	assert.True(t, isContextDone(ctx2))
}

func TestNgrok_NgrokStop_NoMatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	ng := Ngrok{
		NgrokCtx: []NgCtx{
			{
				VMendpoint: "endpoint1",
				CtxCancel:  cancel,
				Ctx:        ctx,
			},
		},
	}

	ng.NgrokStop("nonexistent")

	assert.False(t, isContextDone(ctx))
}

// Helper function to check if context is done
func isContextDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
