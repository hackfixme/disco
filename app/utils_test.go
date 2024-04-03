package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"io"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mr-tron/base58"
	"golang.org/x/crypto/nacl/box"

	actx "go.hackfix.me/disco/app/context"
	"go.hackfix.me/disco/crypto"
	"go.hackfix.me/disco/db/models"
)

type testApp struct {
	*App
	stdin          io.Writer
	stdout, stderr *hookWriter
	env            *mockEnv
	flushOutputs   func() error
}

func newTestApp(ctx context.Context, options ...Option) (*testApp, error) {
	// A unique name per app, to avoid clashing of in-memory SQLite DBs.
	rndName := make([]byte, 12)
	_, err := rand.Read(rndName)
	if err != nil {
		return nil, err
	}

	// Not using just :memory: to avoid 'no such table' issue.
	// See https://github.com/mattn/go-sqlite3#faq
	d, err := initDB(ctx,
		fmt.Sprintf("file:disco-%x?mode=memory&cache=shared", rndName))
	if err != nil {
		return nil, err
	}

	localUser, err := createLocalUser()
	if err != nil {
		return nil, err
	}

	s, err := initKVStore(ctx,
		fmt.Sprintf("file:store-%x?mode=memory&cache=shared", rndName),
		localUser.PrivateKey)
	if err != nil {
		return nil, err
	}

	var (
		stdinR, stdinW   = io.Pipe()
		stdoutW, stderrW = newHookWriter(ctx), newHookWriter(ctx)
	)

	env := &mockEnv{env: map[string]string{}}
	opts := []Option{
		WithContext(ctx),
		WithFDs(stdinR, stdoutW, stderrW),
		WithFS(memoryfs.New()),
		WithLogger(false, false),
		WithEnv(env),
		WithDB(d),
		WithStore(s),
		WithUser(localUser),
	}
	opts = append(opts, options...)
	app, err := New("/disco", opts...)
	if err != nil {
		return nil, err
	}

	tapp := &testApp{
		App: app, stdout: stdoutW, stderr: stderrW,
		stdin: stdinW, env: env,
	}
	tapp.flushOutputs = func() error {
		stdoutW.Reset()
		if _, err := stdoutW.ReadFrom(stdoutW.tmp); err != nil {
			return err
		}
		stdoutW.tmp.Reset()

		stderrW.Reset()
		if _, err := stderrW.ReadFrom(stderrW.tmp); err != nil {
			return err
		}
		stderrW.tmp.Reset()

		return nil
	}

	return tapp, nil
}

func (ta *testApp) Run(args ...string) error {
	if err := ta.App.Run(args); err != nil {
		return err
	}

	if err := ta.flushOutputs(); err != nil {
		return err
	}

	return nil
}

type mockEnv struct {
	mx  sync.RWMutex
	env map[string]string
}

var _ actx.Environment = &mockEnv{}

func (me *mockEnv) Get(key string) string {
	me.mx.RLock()
	defer me.mx.RUnlock()
	return me.env[key]
}

func (me *mockEnv) Set(key, val string) error {
	me.mx.Lock()
	defer me.mx.Unlock()
	me.env[key] = val
	return nil
}

// hookWriter is an io.Writer implementation that listens for writes and
// notifies subscribers when specific text is written.
type hookWriter struct {
	*bytes.Buffer               // main buffer read by tests
	tmp           *bytes.Buffer // temp buffer written to during each command
	ctx           context.Context
	w             chan []byte
	mx            sync.RWMutex
	subs          []chan []byte
}

func newHookWriter(ctx context.Context) *hookWriter {
	hw := &hookWriter{
		Buffer: &bytes.Buffer{},
		tmp:    &bytes.Buffer{},
		ctx:    ctx,
		w:      make(chan []byte, 10),
		subs:   make([]chan []byte, 0),
	}

	go func() {
		for {
			select {
			case d := <-hw.w:
				hw.mx.RLock()
				for _, s := range hw.subs {
					s <- d
				}
				hw.mx.RUnlock()
			case <-hw.ctx.Done():
				return
			}
		}
	}()

	return hw
}

// waitFor starts a goroutine that listens to written data and writes to wCh
// if there's a match of the provided regex pattern.
// If matchIdx > 0, it writes the matched element at that index. This is useful
// for returning substrings.
func (hw *hookWriter) waitFor(rxPat string, matchIdx int, wCh chan string) {
	rx := regexp.MustCompile(rxPat)

	ch := make(chan []byte)
	hw.mx.Lock()
	hw.subs = append(hw.subs, ch)
	hw.mx.Unlock()

	go func() {
		for {
			select {
			case d := <-ch:
				match := rx.FindStringSubmatch(string(d))
				if len(match)-1 >= matchIdx {
					wCh <- match[matchIdx]
					return
				}
			case <-hw.ctx.Done():
				return
			}
		}
	}()
}

func (hw *hookWriter) Write(p []byte) (n int, err error) {
	n, err = hw.tmp.Write(p)
	if err != nil {
		return
	}
	select {
	case hw.w <- p:
	case <-hw.ctx.Done():
	}
	return
}

func createLocalUser() (*models.User, error) {
	pubKey, privKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed generating encryption key pair: %w", err)
	}

	var privKeyHashEnc sql.Null[string]
	privKeyHash := crypto.Hash("encryption key hash", privKey[:])
	privKeyHashEnc.V = base58.Encode(privKeyHash)
	privKeyHashEnc.Valid = true

	localUser := &models.User{
		ID:   1,
		Name: "local",
		Type: models.UserTypeLocal,
		Roles: []*models.Role{
			{
				Name: "admin",
				Permissions: []models.Permission{
					{
						Namespaces: map[string]struct{}{"*": {}},
						Actions:    map[models.Action]struct{}{models.ActionAny: {}},
						Target:     models.PermissionTarget{Resource: models.ResourceAny},
					},
				},
			},
		},
		PublicKey:         pubKey,
		PrivateKey:        privKey,
		PrivateKeyHashEnc: privKeyHashEnc,
	}

	return localUser, nil
}

// newTestContext returns a context that times out after timeout, and an
// assertion handling function that cancels the context prematurely and fails
// the test if the assertion fails. This is done to avoid waiting for the
// context timeout to be reached.
func newTestContext(t *testing.T, timeout time.Duration) (
	ctx context.Context, cancelCtx func(), assertHandler func(bool),
) {
	ctx, cancelCtx = context.WithTimeout(context.Background(), timeout)
	assertHandler = func(success bool) {
		if !success {
			cancelCtx()
			t.FailNow()
		}
	}

	return
}
