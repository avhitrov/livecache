package livecache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var testDateFormat = "2006-01-02"

func TestHeap(t *testing.T) {
	source := prepareData()
	heap := NewHeap(7)
	for _, src := range source {
		heap.Add(src)
	}
	require.Equal(t, 7, heap.MaxElements)
	require.Equal(t, 7, len(heap.Heap))
	require.Equal(t, "3", heap.Heap[0].Key)
	require.Equal(t, "1", heap.Heap[1].Key)
	require.Equal(t, "6", heap.Heap[2].Key)
	require.Equal(t, "9", heap.Heap[3].Key)
	require.Equal(t, "5", heap.Heap[4].Key)
	require.Equal(t, "10", heap.Heap[5].Key)
	require.Equal(t, "7", heap.Heap[6].Key)
}

func prepareData() []*ExpiredKey {
	dtStrArr := []string{
		"2020-01-01",
		"2021-05-03",
		"2020-04-02",
		"2035-04-01",
		"2019-10-11",
		"2019-12-01",
		"2016-10-10",
		"2040-01-01",
		"2015-04-05",
		"2002-01-01",
	}
	keysArr := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}

	res := make([]*ExpiredKey, 0, len(dtStrArr))
	for i := 0; i < len(dtStrArr); i++ {
		dt, err := time.Parse(testDateFormat, dtStrArr[i])
		if err != nil {
			return nil
		}
		res = append(res, &ExpiredKey{Key: keysArr[i], LastAccessed: dt})
	}

	return res
}
