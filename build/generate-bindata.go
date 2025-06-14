// Copyright 2020 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build ignore

package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/klauspost/compress/zstd"
)

func needsUpdate(dir, filename string) (bool, []byte) {
	needRegen := false
	_, err := os.Stat(filename)
	if err != nil {
		needRegen = true
	}

	oldHash, err := os.ReadFile(filename + ".hash")
	if err != nil {
		oldHash = []byte{}
	}

	hasher := sha256.New()

	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		_, _ = hasher.Write([]byte(d.Name()))
		_, _ = hasher.Write([]byte(info.ModTime().String()))
		_, _ = hasher.Write([]byte(strconv.FormatInt(info.Size(), 16)))
		return nil
	})
	if err != nil {
		return true, oldHash
	}

	newHash := hasher.Sum([]byte{})

	if !bytes.Equal(oldHash, newHash) {
		return true, newHash
	}

	return needRegen, newHash
}

func main() {
	if len(os.Args) < 4 {
		log.Fatal("Insufficient number of arguments. Need: directory packageName filename")
	}

	dir, packageName, filename := os.Args[1], os.Args[2], os.Args[3]
	var useGlobalModTime bool
	if len(os.Args) == 5 {
		useGlobalModTime, _ = strconv.ParseBool(os.Args[4])
	}

	update, newHash := needsUpdate(dir, filename)

	if !update {
		fmt.Printf("bindata for %s already up-to-date\n", packageName)
		return
	}

	fmt.Printf("generating bindata for %s\n", packageName)

	root, err := os.OpenRoot(dir)
	if err != nil {
		log.Fatal(err)
	}

	out, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	if err := generate(root.FS(), packageName, useGlobalModTime, out); err != nil {
		log.Fatal(err)
	}
	_ = os.WriteFile(filename+".hash", newHash, 0o666)
}

type file struct {
	Path             string
	Name             string
	UncompressedSize int
	CompressedData   []byte
	UncompressedData []byte
}

type direntry struct {
	Name  string
	IsDir bool
}

func generate(fsRoot fs.FS, packageName string, globalTime bool, output io.Writer) error {
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return err
	}

	files := []file{}

	dirs := map[string][]direntry{}

	if err := fs.WalkDir(fsRoot, ".", func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			entries, err := fs.ReadDir(fsRoot, filePath)
			if err != nil {
				return err
			}
			dirEntries := make([]direntry, 0, len(entries))
			for _, entry := range entries {
				dirEntries = append(dirEntries, direntry{Name: entry.Name(), IsDir: entry.IsDir()})
			}
			dirs[filePath] = dirEntries
			return nil
		}

		src, err := fs.ReadFile(fsRoot, filePath)
		if err != nil {
			return err
		}

		dst := enc.EncodeAll(src, nil)
		if len(dst) < len(src) {
			files = append(files, file{
				Path:             filePath,
				Name:             path.Base(filePath),
				UncompressedSize: len(src),
				CompressedData:   dst,
			})
		} else {
			files = append(files, file{
				Path:             filePath,
				Name:             path.Base(filePath),
				UncompressedData: src,
			})
		}
		return nil
	}); err != nil {
		return err
	}

	return generatedTmpl.Execute(output, map[string]any{
		"Packagename": packageName,
		"GlobalTime":  globalTime,
		"Files":       files,
		"Dirs":        dirs,
	})
}

var generatedTmpl = template.Must(template.New("").Parse(`// Code generated by efs-gen. DO NOT EDIT.

//go:build bindata

package {{.Packagename}}

import (
	"bytes"
	"time"
	"io"
	"io/fs"

	"github.com/klauspost/compress/zstd"
)

type normalFile struct {
	name    string
	content []byte
}

type compressedFile struct {
	name             string
	uncompressedSize int64
	data             []byte
}

var files = map[string]any{
{{- range .Files}}
	"{{.Path}}": {{if .CompressedData}}compressedFile{"{{.Name}}", {{.UncompressedSize}}, []byte({{printf "%+q" .CompressedData}})}{{else}}normalFile{"{{.Name}}", []byte({{printf "%+q" .UncompressedData}})}{{end}},
{{- end}}
}

var dirs = map[string][]fs.DirEntry{
{{- range $key, $entry := .Dirs}}
	"{{$key}}": {
{{- range $entry}}
		direntry{"{{.Name}}", {{.IsDir}}},
{{- end}}
	},
{{- end}}
}

type assets struct{}

var Assets = assets{}

func (a assets) Open(name string) (fs.File, error) {
	f, ok := files[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	switch f := f.(type) {
	case normalFile:
		return file{name: f.name, size: int64(len(f.content)), data: bytes.NewReader(f.content)}, nil
	case compressedFile:
		r, _ := zstd.NewReader(bytes.NewReader(f.data))
		return &compressFile{name: f.name, size: f.uncompressedSize, data: r, content: f.data}, nil
	default:
		panic("unknown file type")
	}
}

func (a assets) ReadDir(name string) ([]fs.DirEntry, error) {
	d, ok := dirs[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return d, nil
}

type file struct {
	name string
	size int64
	data io.ReadSeeker
}

var _ io.ReadSeeker = (*file)(nil)

func (f file) Stat() (fs.FileInfo, error) {
	return fileinfo{name: f.name, size: f.size}, nil
}

func (f file) Read(p []byte) (int, error) {
	return f.data.Read(p)
}

func (f file) Seek(offset int64, whence int) (int64, error) {
	return f.data.Seek(offset, whence)
}

func (f file) Close() error { return nil }

type compressFile struct {
	name    string
	size    int64
	data    *zstd.Decoder
	content []byte
	zstdPos int64
	seekPos int64
}

var _ io.ReadSeeker = (*compressFile)(nil)

func (f *compressFile) Stat() (fs.FileInfo, error) {
	return fileinfo{name: f.name, size: f.size}, nil
}

func (f *compressFile) Read(p []byte) (int, error) {
	if f.zstdPos > f.seekPos {
		if err := f.data.Reset(bytes.NewReader(f.content)); err != nil {
			return 0, err
		}
		f.zstdPos = 0
	}
	if f.zstdPos < f.seekPos {
		if _, err := io.CopyN(io.Discard, f.data, f.seekPos - f.zstdPos); err != nil {
			return 0, err
		}
		f.zstdPos = f.seekPos
	}
	n, err := f.data.Read(p)
	f.zstdPos += int64(n)
	f.seekPos = f.zstdPos
	return n, err
}

func (f *compressFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		f.seekPos = 0 + offset
	case io.SeekCurrent:
		f.seekPos += offset
	case io.SeekEnd:
		f.seekPos = f.size + offset
	}
	return f.seekPos, nil
}

func (f *compressFile) Close() error {
	f.data.Close()
	return nil
}

func (f *compressFile) ZstdBytes() []byte { return f.content }

type fileinfo struct {
	name string
	size int64
}

func (f fileinfo) Name() string       { return f.name }
func (f fileinfo) Size() int64        { return f.size }
func (f fileinfo) Mode() fs.FileMode  { return 0o444 }
func (f fileinfo) ModTime() time.Time { return {{if .GlobalTime}}GlobalModTime(f.name){{else}}time.Unix(0, 0){{end}} }
func (f fileinfo) IsDir() bool        { return false }
func (f fileinfo) Sys() any           { return nil }

type direntry struct {
	name  string
	isDir bool
}

func (d direntry) Name() string { return d.name }
func (d direntry) IsDir() bool  { return d.isDir }
func (d direntry) Type() fs.FileMode {
	if d.isDir {
		return 0o755 | fs.ModeDir
	}
	return 0o444
}
func (direntry) Info() (fs.FileInfo, error) { return nil, fs.ErrNotExist }
`))
