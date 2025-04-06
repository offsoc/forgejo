// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package log

import (
	"bytes"
)

type EventWriterBuffer struct {
	*EventWriterBaseImpl
	Buffer *bytes.Buffer
}

var _ EventWriter = (*EventWriterBuffer)(nil)

func NewEventWriterBuffer(name string, mode WriterMode) *EventWriterBuffer {
	w := &EventWriterBuffer{EventWriterBaseImpl: NewEventWriterBase(name, "buffer", mode)}
	w.Buffer = new(bytes.Buffer)
	w.OutputWriteCloser = nopCloser{w.Buffer}
	return w
}
