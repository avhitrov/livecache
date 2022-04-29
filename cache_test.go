package livecache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// nolint:paralleltest
func TestCacheItem(t *testing.T) {
	ctx := context.Background()
	val1 := "12345-67890-12345-67890"
	cache1 := NewCacheItem(
		func(ctx context.Context) (interface{}, error) {
			a := val1

			return a, nil
		},
		time.Second,
	)

	val2 := "09876-54321-09876-54321"
	cache2 := NewCacheItem(
		func(ctx context.Context) (interface{}, error) {
			a := val2

			return a, nil
		},
		time.Second,
	)

	res1, err := cache1.Get(ctx)
	require.Nil(t, err)

	res2, err := cache2.Get(ctx)
	require.Nil(t, err)

	require.Equal(t, res1.(string), "12345-67890-12345-67890")
	require.Equal(t, res2.(string), "09876-54321-09876-54321")
}

// nolint:paralleltest
func TestCacheBucket(t *testing.T) {
	ctx := context.Background()
	bucket := NewCacheBucket(time.Second, nil, 0)
	val1 := "12345-67890-12345-67890"
	res1, err := bucket.Get(ctx, val1,
		func(ctx context.Context) (interface{}, error) {
			a := val1

			return a, nil
		},
	)
	require.Nil(t, err)

	val2 := "09876-54321-09876-54321"
	res2, err := bucket.Get(ctx, val2,
		func(ctx context.Context) (interface{}, error) {
			a := val2

			return a, nil
		},
	)
	require.Nil(t, err)

	require.Equal(t, res1.(string), "12345-67890-12345-67890")
	require.Equal(t, res2.(string), "09876-54321-09876-54321")
}
