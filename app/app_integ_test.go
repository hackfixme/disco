package app

import (
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test the scenario of 2 Disco nodes, where one creates a user and invitation
// token, and the other joins and reads a remote key over the network.
func TestAppUserInviteJoin(t *testing.T) {
	t.Parallel()

	// wg.Wait must be deferred before the test context cancellation (so that
	// it's called after it when the function returns) to avoid waiting for the
	// context timeout to be reached.
	var wg sync.WaitGroup
	defer wg.Wait()

	timeout := 5 * time.Second
	tctx, cancel, h := newTestContext(t, timeout)
	defer cancel()

	// app1 will accept remote connections
	app1, err := newTestApp(tctx)
	h(assert.NoError(t, err))

	err = app1.Run("init")
	h(assert.NoError(t, err))

	err = app1.Run("set", "key", "testvalue")
	h(assert.NoError(t, err))

	err = app1.Run("get", "key")
	h(assert.NoError(t, err))
	h(assert.Equal(t, "testvalue", app1.stdout.String()))

	err = app1.Run("user", "add", "newuser", "--roles=admin")
	h(assert.NoError(t, err))

	err = app1.Run("invite", "user", "newuser", "--ttl=1m")
	h(assert.NoError(t, err))

	// Extract the invite token from the output
	tokenRx := regexp.MustCompile(`^Token: (.*)\n`)
	match := tokenRx.FindStringSubmatch(app1.stdout.String())
	h(assert.Lenf(t, match, 2, "token not found in output:\n%s", app1.stdout.String()))

	token := match[1]

	// Start the web server on app1
	addrCh := make(chan string)
	app1.stderr.waitFor(`started web server.*address=(.*)\n`, 1, addrCh)

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = app1.Run("serve", "--address=:0")
		h(assert.NoError(t, err))
	}()

	var srvAddress string
	select {
	case srvAddress = <-addrCh:
	case <-tctx.Done():
		t.Fatalf("timed out after %s", timeout)
	}

	// app2 is the remote client that will join app1 with the generated token
	app2, err := newTestApp(tctx)
	h(assert.NoError(t, err))

	err = app2.Run("init")
	h(assert.NoError(t, err))

	err = app2.Run("remote", "add", "testremote", srvAddress, token)
	h(assert.NoError(t, err))

	// The key doesn't exist for app2 locally...
	err = app2.Run("get", "key")
	h(assert.EqualError(t, err, "key 'key' doesn't exist in the 'default' namespace"))
	h(assert.Equal(t, "", app2.stdout.String()))

	// ... but it does exist in the remote node.
	err = app2.Run("get", "--remote=testremote", "key")
	h(assert.NoError(t, err))
	h(assert.Equal(t, "testvalue", app2.stdout.String()))
}
