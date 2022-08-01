package proxy

import (
	"context"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/coinbase/redisbetween/config"
	"github.com/coinbase/redisbetween/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"testing"
)

func TestUpstreamManagerLocateByNameMissingUpstream(t *testing.T) {
	mgr := NewUpstreamManager()
	defer assert.NoError(t, mgr.Shutdown(context.Background()))

	res, ok := mgr.LookupByName(context.TODO(), uuid.New().String())

	assert.False(t, ok)
	assert.Nil(t, res)
}

func TestUpstreamManagerLocateByAddressMissingUpstream(t *testing.T) {
	mgr := NewUpstreamManager()
	defer assert.NoError(t, mgr.Shutdown(context.Background()))

	res, ok := mgr.LookupByAddress(context.TODO(), uuid.New().String())

	assert.False(t, ok)
	assert.Nil(t, res)
}

func TestUpstreamManagerToAddUpstream(t *testing.T) {
	ctx := context.WithValue(context.WithValue(context.Background(), utils.CtxLogKey, zap.L()), utils.CtxStatsdKey, &statsd.NoOpClient{})

	mgr := NewUpstreamManager()
	defer assert.NoError(t, mgr.Shutdown(ctx))

	u := config.Upstream{Name: uuid.New().String(), Address: utils.RedisHost() + ":7006"}
	err := mgr.Add(ctx, &u)
	assert.NoError(t, err)

	res, ok := mgr.LookupByName(ctx, u.Name)
	assert.True(t, ok)
	assert.Equal(t, u.Address, res.Address())

	res, ok = mgr.LookupByAddress(ctx, u.Address)
	assert.True(t, ok)
	assert.Equal(t, u.Address, res.Address())
}

func TestUpstreamManagerToShutdown(t *testing.T) {
	mgr := NewUpstreamManager()

	ctx := context.WithValue(context.WithValue(context.Background(), utils.CtxLogKey, zap.L()), utils.CtxStatsdKey, &statsd.NoOpClient{})
	u := config.Upstream{Name: uuid.New().String(), Address: utils.RedisHost() + ":7006"}
	assert.NoError(t, mgr.Add(ctx, &u))
	res, _ := mgr.LookupByName(ctx, u.Name)

	err := mgr.Shutdown(ctx)

	assert.NoError(t, err)
	assert.Error(t, res.Close(ctx), "server is closed")
}