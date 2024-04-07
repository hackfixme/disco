package app

import (
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAppStore(t *testing.T) {
	t.Parallel()

	tctx, cancel, h := newTestContext(t, 5*time.Second)
	defer cancel()

	app, err := newTestApp(tctx)
	h(assert.NoError(t, err))

	err = app.Run("init")
	h(assert.NoError(t, err))

	t.Run("ok/set_get", func(t *testing.T) {
		err = app.Run("set", "key", "testvalue")
		h(assert.NoError(t, err))

		err = app.Run("set", "key2", "testvalue2")
		h(assert.NoError(t, err))

		err = app.Run("get", "key")
		h(assert.NoError(t, err))
		h(assert.Equal(t, "testvalue", app.stdout.String()))

		err = app.Run("get", "key2")
		h(assert.NoError(t, err))
		h(assert.Equal(t, "testvalue2", app.stdout.String()))
	})

	t.Run("ok/set_get_namespace", func(t *testing.T) {
		err = app.Run("set", "--namespace=dev", "myapp/key", "testvaluens")
		h(assert.NoError(t, err))

		err = app.Run("get", "--namespace=dev", "myapp/key")
		h(assert.NoError(t, err))
		h(assert.Equal(t, "testvaluens", app.stdout.String()))
	})

	t.Run("ok/ls", func(t *testing.T) {
		err = app.Run("ls")
		h(assert.NoError(t, err))
		h(assert.Equal(t, "key\nkey2\n", app.stdout.String()))

		err = app.Run("ls", "--namespace=dev")
		h(assert.NoError(t, err))
		h(assert.Equal(t, "myapp/key\n", app.stdout.String()))

		err = app.Run("ls", "--namespace=*")
		h(assert.NoError(t, err))

		want := "NAMESPACE   KEY       \n" +
			"default     key         \n" +
			"            key2        \n" +
			"dev         myapp/key   \n"
		h(assert.Equal(t, want, app.stdout.String()))
	})

	t.Run("ok/rm_ls", func(t *testing.T) {
		err = app.Run("rm", "key2")
		h(assert.NoError(t, err))

		err = app.Run("rm", "--namespace=dev", "myapp/key")
		h(assert.NoError(t, err))

		err = app.Run("ls", "--namespace=*")
		h(assert.NoError(t, err))

		want := "NAMESPACE   KEY \n" +
			"default     key   \n"
		h(assert.Equal(t, want, app.stdout.String()))
	})

	t.Run("err/missing_key", func(t *testing.T) {
		err = app.Run("get", "missingkey")
		h(assert.EqualError(t, err, "key 'missingkey' doesn't exist in the 'default' namespace"))
	})
}

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
