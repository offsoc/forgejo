// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package ipranges

type container[T any] struct {
	data   []T
	isFull bool
	end    int
}

func newContainer[T any](capacity int) *container[T] {
	return &container[T]{
		data:   make([]T, capacity),
		isFull: false,
		end:    0,
	}
}

func (o *container[T]) all() []T {
	return o.data
}

func (o *container[T]) push(elem T) {
	o.data[o.end] = elem
	o.end++
	if o.end >= len(o.data) {
		o.isFull = true
		o.end = 0
	}
}

func (o *container[T]) length() int {
	if o.isFull {
		return len(o.data)
	}
	return o.end
}

func (o *container[T]) capacity() int {
	return cap(o.data)
}

func (o *container[T]) copy() *container[T] {
	c := make([]T, o.length())
	start := 0
	if o.isFull {
		start = o.end
	}
	for i := 0; i < o.length(); i++ {
		c = append(c, o.data[(start+i)%len(o.data)])
	}
	end := o.end
	if o.isFull {
		end = len(o.data)
	}
	return &container[T]{
		data: c,
		end:  end,
	}
}

func (o *container[T]) reset() {
	o.isFull = false
	o.end = 0
}
