package livecache

import (
	"context"
	"sync"
	"time"
)

type LiveGetter func(ctx context.Context) (interface{}, error)

type CacheItem struct {
	dataActual   interface{}
	dataPrevious interface{}
	nextRefresh  time.Time
	lastUpdated  time.Time
	lastAccessed time.Time
	mx           sync.RWMutex
	getter       LiveGetter
	ttl          time.Duration
}

// NewCacheItem - создает объект кеша, обновляющийся в background-режиме.
func NewCacheItem(
	getter LiveGetter,
	ttl time.Duration,
) *CacheItem {
	return &CacheItem{
		getter:       getter,
		ttl:          ttl,
		lastAccessed: time.Now(),
	}
}

func (ci *CacheItem) Get(ctx context.Context) (interface{}, error) {
	ci.mx.RLock()
	if ci.dataPrevious != nil {
		prev := ci.dataPrevious
		ci.mx.RUnlock()

		return prev, nil
	}
	ci.mx.RUnlock()

	now := time.Now()
	ci.mx.Lock()
	defer ci.mx.Unlock()

	ci.lastAccessed = now
	if ci.dataActual == nil {
		data, err := ci.getter(ctx)
		if err != nil {
			return nil, err
		}
		ci.update(data, now)
	}
	if ci.dataPrevious == nil && now.After(ci.nextRefresh) {
		ci.dataPrevious = ci.dataActual
		go ci.refresh()
	}

	return ci.dataActual, nil
}

func (ci *CacheItem) LastAccessed() time.Time {
	ci.mx.RLock()
	defer ci.mx.RUnlock()

	return ci.lastAccessed
}

func (ci *CacheItem) IsValid() bool {
	now := time.Now()
	ci.mx.RLock()
	defer ci.mx.RUnlock()

	return now.Before(ci.nextRefresh)
}

func (ci *CacheItem) InRefresh() bool {
	ci.mx.RLock()
	defer ci.mx.RUnlock()

	return ci.dataPrevious != nil
}

func (ci *CacheItem) Invalidate() {
	ci.mx.RLock()
	defer ci.mx.RUnlock()

	if ci.dataPrevious != nil {
		return
	}

	go ci.refresh()
}

func (ci *CacheItem) refresh() {
	ctx, cancelFunc := context.WithTimeout(context.Background(), ci.ttl)
	defer cancelFunc()

	data, err := ci.getter(ctx)
	if err != nil {
		ci.mx.Lock()
		ci.dataPrevious = nil
		ci.mx.Unlock()
	} else {
		now := time.Now()
		ci.mx.Lock()
		ci.update(data, now)
		ci.mx.Unlock()
	}
}

func (ci *CacheItem) update(data interface{}, now time.Time) {
	ci.dataActual = data
	ci.dataPrevious = nil
	ci.lastUpdated = now
	ci.lastAccessed = now
	ci.nextRefresh = now.Add(ci.ttl)
}
