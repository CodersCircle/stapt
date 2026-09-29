package main

import (
	"context"
	"io"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/crypto/ssh"
)

const termEvent = "terminal:output"

type termHub struct {
	mu   sync.Mutex
	byID map[string]*termSession
	seq  int
}

type termSession struct {
	sess  *ssh.Session
	stdin io.WriteCloser
}

func newTermHub() *termHub {
	return &termHub{byID: map[string]*termSession{}}
}

func (h *termHub) Register(sess *ssh.Session, stdin io.WriteCloser) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.seq++
	id := "t" + itoa(h.seq)
	h.byID[id] = &termSession{sess: sess, stdin: stdin}
	return id
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func (h *termHub) pump(ctx context.Context, id string, stdout io.Reader) {
	buf := make([]byte, 8192)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			runtime.EventsEmit(ctx, termEvent, TerminalOutputEvent{
				SessionID: id,
				DataB64:   encodeTermChunk(buf[:n]),
			})
		}
		if err != nil {
			break
		}
	}
	h.Stop(id)
	runtime.EventsEmit(ctx, "terminal:closed", id)
}

func (h *termHub) Write(id, data string) error {
	h.mu.Lock()
	s := h.byID[id]
	h.mu.Unlock()
	if s == nil {
		return io.EOF
	}
	_, err := io.WriteString(s.stdin, data)
	return err
}

func (h *termHub) Resize(id string, cols, rows int) error {
	h.mu.Lock()
	s := h.byID[id]
	h.mu.Unlock()
	if s == nil {
		return io.EOF
	}
	return resizePTY(s.sess, cols, rows)
}

func (h *termHub) Stop(id string) {
	h.mu.Lock()
	s, ok := h.byID[id]
	if ok {
		delete(h.byID, id)
	}
	h.mu.Unlock()
	if !ok || s == nil {
		return
	}
	_ = s.stdin.Close()
	_ = s.sess.Close()
}

func (h *termHub) CloseAll() {
	h.mu.Lock()
	ids := make([]string, 0, len(h.byID))
	for id := range h.byID {
		ids = append(ids, id)
	}
	h.mu.Unlock()
	for _, id := range ids {
		h.Stop(id)
	}
}
