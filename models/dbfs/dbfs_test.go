// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package dbfs

import (
	"bufio"
	"io"
	"os"
	"testing"

	"forgejo.org/models/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func changeDefaultFileBlockSize(n int64) (restore func()) {
	old := defaultFileBlockSize
	defaultFileBlockSize = n
	return func() {
		defaultFileBlockSize = old
	}
}

func TestDbfsBasic(t *testing.T) {
	defer changeDefaultFileBlockSize(4)()

	// test basic write/read
	f, err := OpenFile(db.DefaultContext, "test.txt", os.O_RDWR|os.O_CREATE)
	require.NoError(t, err)

	n, err := f.Write([]byte("0123456789")) // blocks: 0123 4567 89
	require.NoError(t, err)
	assert.Equal(t, 10, n)

	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err)

	buf, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, 10, n)
	assert.Equal(t, "0123456789", string(buf))

	// write some new data
	_, err = f.Seek(1, io.SeekStart)
	require.NoError(t, err)
	_, err = f.Write([]byte("bcdefghi")) // blocks: 0bcd efgh i9
	require.NoError(t, err)

	// read from offset
	buf, err = io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, "9", string(buf))

	// read all
	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err)
	buf, err = io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, "0bcdefghi9", string(buf))

	// write to new size
	_, err = f.Seek(-1, io.SeekEnd)
	require.NoError(t, err)
	_, err = f.Write([]byte("JKLMNOP")) // blocks: 0bcd efgh iJKL MNOP
	require.NoError(t, err)
	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err)
	buf, err = io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, "0bcdefghiJKLMNOP", string(buf))

	// write beyond EOF and fill with zero
	_, err = f.Seek(5, io.SeekCurrent)
	require.NoError(t, err)
	_, err = f.Write([]byte("xyzu")) // blocks: 0bcd efgh iJKL MNOP 0000 0xyz u
	require.NoError(t, err)
	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err)
	buf, err = io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, "0bcdefghiJKLMNOP\x00\x00\x00\x00\x00xyzu", string(buf))

	// write to the block with zeros
	_, err = f.Seek(-6, io.SeekCurrent)
	require.NoError(t, err)
	_, err = f.Write([]byte("ABCD")) // blocks: 0bcd efgh iJKL MNOP 000A BCDz u
	require.NoError(t, err)
	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err)
	buf, err = io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, "0bcdefghiJKLMNOP\x00\x00\x00ABCDzu", string(buf))

	require.NoError(t, f.Close())

	// test rename
	err = Rename(db.DefaultContext, "test.txt", "test2.txt")
	require.NoError(t, err)

	_, err = OpenFile(db.DefaultContext, "test.txt", os.O_RDONLY)
	require.Error(t, err)

	f, err = OpenFile(db.DefaultContext, "test2.txt", os.O_RDONLY)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	// test remove
	err = Remove(db.DefaultContext, "test2.txt")
	require.NoError(t, err)

	_, err = OpenFile(db.DefaultContext, "test2.txt", os.O_RDONLY)
	require.Error(t, err)

	// test stat
	f, err = OpenFile(db.DefaultContext, "test/test.txt", os.O_RDWR|os.O_CREATE)
	require.NoError(t, err)
	stat, err := f.Stat()
	require.NoError(t, err)
	assert.Equal(t, "test.txt", stat.Name())
	assert.EqualValues(t, 0, stat.Size())
	_, err = f.Write([]byte("0123456789"))
	require.NoError(t, err)
	stat, err = f.Stat()
	require.NoError(t, err)
	assert.EqualValues(t, 10, stat.Size())
}

func TestDbfsReadWrite(t *testing.T) {
	defer changeDefaultFileBlockSize(4)()

	f1, err := OpenFile(db.DefaultContext, "test.log", os.O_RDWR|os.O_CREATE)
	require.NoError(t, err)
	defer f1.Close()

	f2, err := OpenFile(db.DefaultContext, "test.log", os.O_RDONLY)
	require.NoError(t, err)
	defer f2.Close()

	_, err = f1.Write([]byte("line 1\n"))
	require.NoError(t, err)

	f2r := bufio.NewReader(f2)

	line, err := f2r.ReadString('\n')
	require.NoError(t, err)
	assert.Equal(t, "line 1\n", line)
	_, err = f2r.ReadString('\n')
	require.ErrorIs(t, err, io.EOF)

	_, err = f1.Write([]byte("line 2\n"))
	require.NoError(t, err)

	line, err = f2r.ReadString('\n')
	require.NoError(t, err)
	assert.Equal(t, "line 2\n", line)
	_, err = f2r.ReadString('\n')
	require.ErrorIs(t, err, io.EOF)
}

func TestDbfsSeekWrite(t *testing.T) {
	defer changeDefaultFileBlockSize(4)()

	f, err := OpenFile(db.DefaultContext, "test2.log", os.O_RDWR|os.O_CREATE)
	require.NoError(t, err)
	defer f.Close()

	n, err := f.Write([]byte("111"))
	require.NoError(t, err)

	_, err = f.Seek(int64(n), io.SeekStart)
	require.NoError(t, err)

	_, err = f.Write([]byte("222"))
	require.NoError(t, err)

	_, err = f.Seek(int64(n), io.SeekStart)
	require.NoError(t, err)

	_, err = f.Write([]byte("333"))
	require.NoError(t, err)

	fr, err := OpenFile(db.DefaultContext, "test2.log", os.O_RDONLY)
	require.NoError(t, err)
	defer f.Close()

	buf, err := io.ReadAll(fr)
	require.NoError(t, err)
	assert.Equal(t, "111333", string(buf))
}
