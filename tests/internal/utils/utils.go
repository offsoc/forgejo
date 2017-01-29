package utils

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// T wraps testing.T and the configurations of the testing instance.
type T struct {
	*testing.T
	Config *Config
}

// New create an instance of T
func New(t *testing.T, c *Config) *T {
	return &T{T: t, Config: c}
}

// Config Settings of the testing program
type Config struct {
	// The executable path of the tested program.
	Program string
	// Working directory prepared for the tested program.
	// If empty, a directory named with random suffixes is picked, and created under the current directory.
	// The directory will be removed when the test finishes.
	WorkDir string
	// Command-line arguments passed to the tested program.
	Args []string

	// Where to redirect the stdout/stderr to. For debugging purposes.
	LogFile *os.File
}

func redirect(cmd *exec.Cmd, f *os.File) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	go io.Copy(os.Stderr, stdout)
	go io.Copy(os.Stdout, stderr)
	return nil
}

// RunTest Helper function for setting up a running Gitea server for functional testing and then gracefully terminating it.
func (t *T) RunTest(tests ...func(*T) error) (err error) {
	if t.Config.Program == "" {
		return errors.New("Need input file")
	}

	path, err := filepath.Abs(t.Config.Program)
	if err != nil {
		return err
	}

	workdir := t.Config.WorkDir
	if workdir == "" {
		workdir, err = filepath.Abs(fmt.Sprintf("%s-%10d", filepath.Base(t.Config.Program), time.Now().UnixNano()))
		if err != nil {
			return err
		}
		if err := os.Mkdir(workdir, 0700); err != nil {
			return err
		}
		defer os.RemoveAll(workdir)
	}

	newpath := filepath.Join(workdir, filepath.Base(path))
	if err := os.Link(path, newpath); err != nil {
		return err
	}

	log.Printf("Starting the server: %s args:%s workdir:%s", newpath, t.Config.Args, workdir)

	cmd := exec.Command(newpath, t.Config.Args...)
	cmd.Dir = workdir

	if t.Config.LogFile != nil {
		if err := redirect(cmd, t.Config.LogFile); err != nil {
			return err
		}
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	log.Println("Server started.")

	defer func() {
		// Do not early return. We have to call Wait anyway.
		_ = cmd.Process.Signal(syscall.SIGTERM)

		if _err := cmd.Wait(); _err != nil {
			if _err.Error() != "signal: terminated" {
				err = _err
				return
			}
		}

		log.Println("Server exited")
	}()

	for _, fn := range tests {
		if err := fn(t); err != nil {
			return err
		}
	}

	return nil
}
