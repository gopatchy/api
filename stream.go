package patchy

import (
	"sync"
)

type GetStream[T any] struct {
	ch  chan *T
	gsi *getStreamInt

	err error

	mu sync.RWMutex
}

func (gs *GetStream[T]) Close() {
	gs.gsi.Close()
}

func (gs *GetStream[T]) Chan() <-chan *T {
	return gs.ch
}

func (gs *GetStream[T]) Read() *T {
	return <-gs.Chan()
}

func (gs *GetStream[T]) Error() error {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	return gs.err
}

func (gs *GetStream[T]) writeEvent(obj *T) {
	gs.ch <- obj
}

func (gs *GetStream[T]) writeError(err error) {
	gs.mu.Lock()
	gs.err = err
	gs.mu.Unlock()

	close(gs.ch)
}

type ListStream[T any] struct {
	ch  chan []*T
	lsi *listStreamInt

	err error

	mu sync.RWMutex
}

func (ls *ListStream[T]) Close() {
	ls.lsi.Close()
}

func (ls *ListStream[T]) Chan() <-chan []*T {
	return ls.ch
}

func (ls *ListStream[T]) Read() []*T {
	return <-ls.Chan()
}

func (ls *ListStream[T]) Error() error {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	return ls.err
}

func (ls *ListStream[T]) writeEvent(list []*T) {
	ls.ch <- list
}

func (ls *ListStream[T]) writeError(err error) {
	ls.mu.Lock()
	ls.err = err
	ls.mu.Unlock()

	close(ls.ch)
}
