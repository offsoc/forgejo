// Copyright 2023 The Gitea Authors.
// SPDX-License-Identifier: MIT

package admin

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"time"

	"forgejo.org/modules/httplib"
	"forgejo.org/services/context"
)

func MonitorDiagnosis(ctx *context.Context) {
	seconds := ctx.FormInt64("seconds")
	seconds = max(5, min(300, seconds))

	httplib.ServeSetHeaders(ctx.Resp, &httplib.ServeHeaderOptions{
		ContentType: "application/zip",
		Disposition: "attachment",
		Filename:    fmt.Sprintf("forgejo-diagnosis-%s.zip", time.Now().Format("20060102-150405")),
	})

	zipWriter := zip.NewWriter(ctx.Resp)
	defer zipWriter.Close()

	f, err := zipWriter.CreateHeader(&zip.FileHeader{Name: "goroutine-before.txt", Method: zip.Deflate, Modified: time.Now()})
	if err != nil {
		ctx.ServerError("Failed to create zip file", err)
		return
	}
	_ = pprof.Lookup("goroutine").WriteTo(f, 1)

	f, err = zipWriter.CreateHeader(&zip.FileHeader{Name: "cpu-profile.dat", Method: zip.Deflate, Modified: time.Now()})
	if err != nil {
		ctx.ServerError("Failed to create zip file", err)
		return
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		_, _ = f.Write([]byte(err.Error()))
	}

	traceBuf := &bytes.Buffer{}
	if err := trace.Start(traceBuf); err != nil {
		_, _ = traceBuf.Write([]byte(err.Error()))
	}

	select {
	case <-time.After(time.Duration(seconds) * time.Second):
	case <-ctx.Done():
	}
	pprof.StopCPUProfile()
	trace.Stop()

	f, err = zipWriter.CreateHeader(&zip.FileHeader{Name: "trace.dat", Method: zip.Deflate, Modified: time.Now()})
	if err != nil {
		ctx.ServerError("Failed to create zip file", err)
		return
	}

	if _, err := io.Copy(f, traceBuf); err != nil {
		ctx.ServerError("Failed to create zip file", err)
		return
	}

	f, err = zipWriter.CreateHeader(&zip.FileHeader{Name: "goroutine-after.txt", Method: zip.Deflate, Modified: time.Now()})
	if err != nil {
		ctx.ServerError("Failed to create zip file", err)
		return
	}
	_ = pprof.Lookup("goroutine").WriteTo(f, 1)

	f, err = zipWriter.CreateHeader(&zip.FileHeader{Name: "heap.dat", Method: zip.Deflate, Modified: time.Now()})
	if err != nil {
		ctx.ServerError("Failed to create zip file", err)
		return
	}
	// To avoid showing memory that actually can be cleaned, run the garbage
	// collector.
	runtime.GC()
	_ = pprof.Lookup("heap").WriteTo(f, 0)
}
