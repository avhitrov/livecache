package livecache

import (
	"context"
	"sync"
	"time"
)

const clearCycleInterval = 100 * time.Millisecond

type CacheBucket struct {
	expireInterval time.Duration
	clearInterval  *time.Duration
	maxElements    int
	mx             sync.RWMutex
	data           map[string]*CacheItem
	stopCleaner    context.CancelFunc
}

type ExpiredKey struct {
	Key          string
	LastAccessed time.Time
}

// NewCacheBucket создает мапу из объектов CacheItem, ограниченных по
// времени использования (если при вызове указано время жизни объектов от последнего доступа
// к содержимому),
// количеству сохраняемых элементов (если при вызове указать max > 0).
// Применение: необходимо хранить пул однородных данных, например,
// результат поиска в зависимости от набора входных параметров или
// пул профилей активных пользователей.
// Хорошим стилем при большом объеме данных является использование нескольких бакетов объемом
// до 10000 элементов с разбиением по ключу.
// ToDo: пробрасывать сюда context приложения, чтобы клинер можно было остановить мягким шатдауном.
func NewCacheBucket(expire time.Duration, clear *time.Duration, max int) *CacheBucket {
	ctx, cancelFunc := context.WithCancel(context.Background())

	cb := CacheBucket{
		expireInterval: expire,
		clearInterval:  clear,
		maxElements:    max,
		stopCleaner:    cancelFunc,
		data:           make(map[string]*CacheItem, max),
	}
	if cb.clearInterval != nil || cb.maxElements > 0 {
		go cb.cleaner(ctx)
	}

	return &cb
}

func (cb *CacheBucket) Get(
	ctx context.Context,
	key string,
	getter LiveGetter,
) (interface{}, error) {
	cb.mx.RLock()
	cacher, ok := cb.data[key]
	cb.mx.RUnlock()

	if !ok {
		cacher = NewCacheItem(getter, cb.expireInterval)
		cb.mx.Lock()
		cb.data[key] = cacher
		cb.mx.Unlock()
	}

	return cacher.Get(ctx)
}

func (cb *CacheBucket) IsValid(key string) bool {
	cb.mx.RLock()
	defer cb.mx.RUnlock()
	cacher, exists := cb.data[key]

	if !exists {
		return false
	}

	return cacher.IsValid()
}

func (cb *CacheBucket) InRefresh(key string) bool {
	cb.mx.RLock()
	defer cb.mx.RUnlock()
	cacher, exists := cb.data[key]

	if !exists {
		return false
	}

	return cacher.InRefresh()
}

func (cb *CacheBucket) Invalidate(key string) {
	cb.mx.RLock()
	defer cb.mx.RUnlock()
	cacher, exists := cb.data[key]

	if !exists {
		return
	}

	cacher.Invalidate()
}

func (cb *CacheBucket) StopCleaner() {
	cb.mx.RLock()
	defer cb.mx.RUnlock()

	cb.stopCleaner()
}

// nolint:nestif
func (cb *CacheBucket) cleaner(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if cb.clearInterval != nil {
				keysToClean := make([]string, 0, 10)
				now := time.Now()

				cb.mx.RLock()
				for key, data := range cb.data {
					lastAccessedCheck := data.LastAccessed().Add(*cb.clearInterval)
					if now.After(lastAccessedCheck) {
						keysToClean = append(keysToClean, key)
					}
				}
				cb.mx.RUnlock()
				cb.clearItems(keysToClean)
			}
			time.Sleep(clearCycleInterval)

			if cb.maxElements > 0 {
				var heap *LatestHeap
				cb.mx.RLock()
				exceed := len(cb.data) - cb.maxElements
				if exceed > 0 {
					heap = NewHeap(exceed)
					for key, data := range cb.data {
						heap.Add(&ExpiredKey{key, data.lastAccessed})
					}
				}
				cb.mx.RUnlock()

				if heap != nil {
					keysToClean := make([]string, 0, len(heap.Heap))
					for _, data := range heap.Heap {
						keysToClean = append(keysToClean, data.Key)
					}
					if len(keysToClean) > 0 {
						cb.clearItems(keysToClean)
					}
				}
			}
			time.Sleep(clearCycleInterval)
		}
	}
}

func (cb *CacheBucket) clearItems(keys []string) {
	cb.mx.Lock()
	defer cb.mx.Unlock()

	for _, key := range keys {
		delete(cb.data, key)
	}
}
