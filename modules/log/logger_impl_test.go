// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testGeneric[T any](log *LoggerImpl, t T) {
	log.Log(0, INFO, "Just testing the logging of a generic function %v", t)
}

func TestLog(t *testing.T) {
	bufferWriter := NewEventWriterBuffer("test-buffer", WriterMode{
		Level: INFO,
	})

	logger := NewLoggerWithWriters(t.Context(), "test", bufferWriter)

	testGeneric(logger, "I'm the generic value!")
	logger.Close()

	assert.Contains(t, bufferWriter.Buffer.String(), ".../logger_impl_test.go:13:testGeneric() [I] Just testing the logging of a generic function I'm the generic value!")
}
