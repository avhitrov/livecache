package livecache

// Куча для сбора претендентов на удаление.
// Сортируется по дате последнего доступа. Самый последний по времени элемент находится в вершине,
// ниже располагаются более старые. Служит для определения списка элементов для удаления из бакета.
// Понижает вычислительную сложность алгоритма с O(n2) до O(n*log(n)).
type LatestHeap struct {
	Heap        []*ExpiredKey
	MaxElements int
}

func NewHeap(max int) *LatestHeap {
	heap := &LatestHeap{MaxElements: max}
	if heap.MaxElements > 0 {
		heap.Heap = make([]*ExpiredKey, 0, heap.MaxElements)
	}

	return heap
}

// Add встраивает в кучу элемент.:
// - если размер кучи не достиг MaxElements, то элемент просто добавляет в кучу в свое место;
// - если размер кучи достиг MaxElements, то элемент добавляется в вершину (если он более старый,
// чем на вершине) и производится пересчет кучи (Heapify).
func (lh *LatestHeap) Add(newElement *ExpiredKey) {
	if lh.MaxElements == 0 {
		return
	}
	if len(lh.Heap) < lh.MaxElements {
		lh.Heap = append(lh.Heap, newElement)
		lh.UpHeap(len(lh.Heap) - 1)
	} else if newElement.LastAccessed.Before(lh.Heap[0].LastAccessed) {
		lh.Heap[0] = newElement
		lh.Heapify(0)
	}
}

// UpHeap добавляет элемент в кучу.
func (lh *LatestHeap) UpHeap(index int) {
	if index == 0 {
		return
	}

	parentIndex := (index - 1) / 2

	if !lh.Heap[parentIndex].LastAccessed.Before(lh.Heap[index].LastAccessed) {
		return
	}

	lh.Heap[parentIndex], lh.Heap[index] = lh.Heap[index], lh.Heap[parentIndex]

	lh.UpHeap(parentIndex)
}

// Heapify заново сортирует кучу от вершины после добавления в нее элемента.
func (lh *LatestHeap) Heapify(index int) {
	i, v := lh.LatestChild(index)
	if i == -1 {
		return
	}
	if lh.Heap[index].LastAccessed.After(v.LastAccessed) {
		return
	}
	lh.Heap[i], lh.Heap[index] = lh.Heap[index], lh.Heap[i]
	lh.Heapify(i)
}

// LatestChild сравнивает текущий элемент с обоими потомками и возвращает
// самый старый по LastAccessed элемент и его индекс.
func (lh *LatestHeap) LatestChild(index int) (i int, v *ExpiredKey) {
	if 2*index+1 >= len(lh.Heap) {
		return -1, nil
	}
	if 2*index+2 >= len(lh.Heap) {
		return 2*index + 1, lh.Heap[2*index+1]
	}
	if lh.Heap[2*index+2].LastAccessed.Before(lh.Heap[2*index+1].LastAccessed) {
		return 2*index + 1, lh.Heap[2*index+1]
	}

	return 2*index + 2, lh.Heap[2*index+2]
}
