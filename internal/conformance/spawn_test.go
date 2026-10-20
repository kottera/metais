// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// spawnerName is the name of testdata/spawner, the program that starts
// every process the runner runs where Go starts a process by fork(2).
const spawnerName = "metais-spawner"

// spawnerPath is the spawner TestMain built, or "" where the runner
// starts each process itself.
var spawnerPath string

// spawnRequest and spawnResponse are the spawner's protocol: its request
// and response have the same fields (testdata/spawner).
type spawnRequest struct {
	ID    uint64
	Path  string
	Args  []string
	Dir   string
	Env   []string
	Stdin []byte
	Wall  time.Duration
	Cap   int
}

type spawnResponse struct {
	ID       uint64
	StartErr string
	RunErr   string
	Stdout   []byte
	Stderr   []byte
	Over     bool
	Exit     int
	CPU      time.Duration
	RSS      int64
	RSSOK    bool
	Killed   bool
}

// spawnClient is the runner's side of one running spawner: requests go
// out under mu, and one goroutine hands each response to the call
// waiting for its ID.
type spawnClient struct {
	cmd     *exec.Cmd
	in      io.WriteCloser
	mu      sync.Mutex
	enc     *gob.Encoder
	next    uint64
	waiting map[uint64]chan *spawnResponse
	err     error // why the spawner can serve no more requests
}

var (
	spawnOnce sync.Once
	spawnc    *spawnClient
	spawnErr  error
)

// spawner returns the running spawner, starting it the first time, or
// nil where there is none.
func spawner() (*spawnClient, error) {
	if spawnerPath == "" {
		return nil, nil
	}
	spawnOnce.Do(func() { spawnc, spawnErr = startSpawner(spawnerPath) })
	return spawnc, spawnErr
}

// startSpawner starts the spawner at path, in this process's working
// directory and environment, so that it finds a program by name as this
// process would.
func startSpawner(path string) (*spawnClient, error) {
	cmd := exec.Command(path)
	cmd.Stderr = os.Stderr
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting the spawner: %w", err)
	}
	c := &spawnClient{cmd: cmd, in: in, enc: gob.NewEncoder(in), waiting: map[uint64]chan *spawnResponse{}}
	go c.read(gob.NewDecoder(out))
	return c, nil
}

// read hands each response to the call waiting for it, until the
// spawner's output ends; then it fails every call still waiting.
func (c *spawnClient) read(dec *gob.Decoder) {
	for {
		resp := new(spawnResponse)
		if err := dec.Decode(resp); err != nil {
			c.mu.Lock()
			c.err = fmt.Errorf("the spawner: %w", err)
			for id, ch := range c.waiting {
				close(ch)
				delete(c.waiting, id)
			}
			c.mu.Unlock()
			return
		}
		c.mu.Lock()
		ch := c.waiting[resp.ID]
		delete(c.waiting, resp.ID)
		c.mu.Unlock()
		if ch != nil {
			ch <- resp
		}
	}
}

// run runs path through the spawner as runProc runs it directly, and
// returns what runProc returns.
func (c *spawnClient) run(path string, args []string, dir string, env []string, stdin []byte, wall time.Duration) (*proc, error) {
	ch := make(chan *spawnResponse, 1)
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return nil, c.err
	}
	c.next++
	req := spawnRequest{ID: c.next, Path: path, Args: args, Dir: dir, Env: env, Stdin: stdin, Wall: wall, Cap: maxInflate}
	c.waiting[req.ID] = ch
	if err := c.enc.Encode(&req); err != nil {
		delete(c.waiting, req.ID)
		c.err = fmt.Errorf("the spawner: %w", err)
		c.mu.Unlock()
		return nil, c.err
	}
	c.mu.Unlock()
	resp, ok := <-ch
	if !ok {
		c.mu.Lock()
		defer c.mu.Unlock()
		return nil, c.err
	}
	if resp.StartErr != "" {
		return nil, fmt.Errorf("start %s: %s", path, resp.StartErr)
	}
	if resp.Over {
		return nil, fmt.Errorf("%s wrote more than the runner's %d bytes to a standard stream", path, maxInflate)
	}
	if resp.RunErr != "" {
		return nil, fmt.Errorf("run %s: %s", path, resp.RunErr)
	}
	p := &proc{stdout: resp.Stdout, stderr: resp.Stderr, exit: resp.Exit, cpu: resp.CPU, rss: -1, killed: resp.Killed}
	if resp.RSSOK {
		p.rss = maxRSSBytes(runtime.GOOS, resp.RSS)
	}
	return p, nil
}

// stop ends the spawner: its input ends, so it kills what is still
// running, and it is waited for.
func (c *spawnClient) stop() error {
	c.in.Close()
	return c.cmd.Wait()
}

// buildSpawner builds testdata/spawner, without the race detector, into
// dir, and returns its path.
func buildSpawner(dir string) (string, error) {
	out := filepath.Join(dir, spawnerName)
	cmd := exec.Command("go", "build", "-trimpath", "-o", out, "./internal/conformance/testdata/spawner")
	cmd.Dir = moduleRoot
	if b, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("building the spawner: %v\n%s", err, b)
	}
	return out, nil
}
