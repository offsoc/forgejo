// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"iter"
	"maps"
)

type Set[T comparable] map[T]struct{}

// SetOf creates a set and adds the specified elements to it.
func SetOf[T comparable](values ...T) Set[T] {
	s := make(Set[T], len(values))
	s.AddMultiple(values...)
	return s
}

// Add adds the specified element to a set.
// Returns true if the element is added; false if the element is already present.
func (s Set[T]) Add(value T) bool {
	if _, has := s[value]; !has {
		s[value] = struct{}{}
		return true
	}
	return false
}

// AddMultiple adds the specified elements to a set.
func (s Set[T]) AddMultiple(values ...T) {
	for _, value := range values {
		s.Add(value)
	}
}

func (s Set[T]) IsSubset(subset []T) bool {
	for _, v := range subset {
		if !s.Contains(v) {
			return false
		}
	}
	return true
}

// Contains determines whether a set contains the specified element.
// Returns true if the set contains the specified element; otherwise, false.
func (s Set[T]) Contains(value T) bool {
	_, has := s[value]
	return has
}

// Remove removes the specified element.
// Returns true if the element is successfully found and removed; otherwise, false.
func (s Set[T]) Remove(value T) bool {
	if _, has := s[value]; has {
		delete(s, value)
		return true
	}
	return false
}

// Values gets a list of all elements in the set.
func (s Set[T]) Values() []T {
	keys := make([]T, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	return keys
}

// Seq returns a iterator over the elements in the set.
// It returns a single-use iterator.
func (s Set[T]) Seq() iter.Seq[T] {
	return maps.Keys(s)
}
