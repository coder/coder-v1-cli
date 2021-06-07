package wsnet

import (
	"fmt"
	"testing"

	"cdr.dev/slog/sloggers/slogtest/assert"
)

func Test_BrokerMessage(t *testing.T) {
	t.Run("getAddress", func(t *testing.T) {
		t.Run("OK", func(t *testing.T) {
			var (
				msg = BrokerMessage{
					Policies: nil,
				}
				network = "tcp"
				addr    = "localhost:1234"
			)

			protocol := formatAddress(network, addr)
			gotNetwork, gotAddr, err := msg.getAddress(protocol)
			assert.Success(t, "got address", err)
			assert.Equal(t, "networks equal", network, gotNetwork)
			assert.Equal(t, "addresses equal", addr, gotAddr)

			msg.Policies = []DialPolicy{}
			gotNetwork, gotAddr, err = msg.getAddress(protocol)
			assert.Success(t, "got address", err)
			assert.Equal(t, "networks equal", network, gotNetwork)
			assert.Equal(t, "addresses equal", addr, gotAddr)
		})

		t.Run("InvalidProtocol", func(t *testing.T) {
			cases := []struct {
				protocol    string
				errContains string
			}{
				{
					protocol:    "",
					errContains: "invalid",
				},
				{
					protocol:    "a:b",
					errContains: "invalid",
				},
				{
					protocol:    "a:b:c:d",
					errContains: "invalid",
				},
				{
					protocol:    ":localhost:1234",
					errContains: "network",
				},
				{
					protocol:    "tcp::1234",
					errContains: "host",
				},
				{
					protocol:    "tcp:localhost:",
					errContains: "port",
				},
				{
					protocol:    "tcp:localhost:asdf",
					errContains: "port",
				},
				{
					protocol:    "tcp:localhost:-1",
					errContains: "port",
				},
				{
					// Overflow uint16.
					protocol:    fmt.Sprintf("tcp:localhost:%v", uint(1)<<16),
					errContains: "port",
				},
			}

			var msg BrokerMessage
			for i, c := range cases {
				amsg := fmt.Sprintf("case %v %q: ", i, c)
				gotNetwork, gotAddr, err := msg.getAddress(c.protocol)
				assert.Error(t, amsg+"successfully got invalid address", err)
				assert.ErrorContains(t, fmt.Sprintf("%verr contains %q", amsg, c.errContains), err, c.errContains)
				assert.Equal(t, amsg+"empty network", "", gotNetwork)
				assert.Equal(t, amsg+"empty address", "", gotAddr)
			}
		})

		t.Run("ChecksPolicies", func(t *testing.T) {
			// ok == true tests automatically have a bunch of non-matching dial
			// policies injected in front of them.
			cases := []struct {
				network string
				host    string
				port    uint16
				policy  DialPolicy
				ok      bool
			}{
				{
					network: "tcp",
					host:    "localhost",
					port:    1234,
					policy:  dialPolicy("tcp", "localhost", 1234),
					ok:      true,
				},
				{
					network: "tcp",
					host:    "localhost",
					port:    1234,
					policy:  dialPolicy("udp", "example.com", 51),
					ok:      false,
				},
				// Network checks.
				{
					network: "tcp",
					host:    "localhost",
					port:    1234,
					policy:  dialPolicy("", "localhost", 1234),
					ok:      true,
				},
				{
					network: "tcp",
					host:    "localhost",
					port:    1234,
					policy:  dialPolicy("udp", "localhost", 1234),
					ok:      false,
				},
				// Host checks.
				{
					network: "tcp",
					host:    "localhost",
					port:    1234,
					policy:  dialPolicy("tcp", "", 1234),
					ok:      true,
				},
				{
					network: "tcp",
					host:    "localhost",
					port:    1234,
					policy:  dialPolicy("tcp", "127.0.0.1", 1234),
					ok:      true,
				},
				{
					network: "tcp",
					host:    "127.0.0.1",
					port:    1234,
					policy:  dialPolicy("tcp", "127.1.2.3", 1234),
					ok:      true,
				},
				{
					network: "tcp",
					host:    "[::1]",
					port:    1234,
					policy:  dialPolicy("tcp", "127.1.2.3", 1234),
					ok:      true,
				},
				{
					network: "tcp",
					host:    "localhost",
					port:    1234,
					policy:  dialPolicy("tcp", "example.com", 1234),
					ok:      false,
				},
				{
					network: "tcp",
					host:    "example.com",
					port:    1234,
					policy:  dialPolicy("tcp", "localhost", 1234),
					ok:      false,
				},
				// Port checks.
				{
					network: "tcp",
					host:    "localhost",
					port:    1234,
					policy:  dialPolicy("tcp", "localhost", 5678),
					ok:      false,
				},
				{
					network: "tcp",
					host:    "localhost",
					port:    1234,
					policy:  dialPolicy("tcp", "localhost", 0),
					ok:      true,
				},
			}

			for i, c := range cases {
				var (
					amsg = fmt.Sprintf("case %v '%+v': ", i, c)
					msg  = BrokerMessage{
						Policies: []DialPolicy{c.policy},
					}
				)

				// Add nonsense policies before the matching policy.
				if c.ok {
					msg.Policies = []DialPolicy{
						dialPolicy("asdf", "localhost", 1234),
						dialPolicy("tcp", "asdf", 1234),
						dialPolicy("tcp", "localhost", 17208),
						c.policy,
					}
				}

				// Test DialPolicy.
				assert.Equal(t, amsg+"policy matches", c.ok, c.policy.permits(c.network, canonicalizeHost(c.host), c.port))

				// Test BrokerMessage.
				protocol := formatAddress(c.network, fmt.Sprintf("%v:%v", c.host, c.port))
				gotNetwork, gotAddr, err := msg.getAddress(protocol)
				if c.ok {
					assert.Success(t, amsg, err)
				} else {
					assert.Error(t, amsg+"successfully got invalid address", err)
					assert.ErrorContains(t, amsg+"err contains 'not permitted'", err, "not permitted")
					assert.Equal(t, amsg+"empty network", "", gotNetwork)
					assert.Equal(t, amsg+"empty address", "", gotAddr)
				}
			}
		})
	})
}

func formatAddress(network, addr string) string {
	return fmt.Sprintf("%v:%v", network, addr)
}

func dialPolicy(network, host string, port uint16) DialPolicy {
	return DialPolicy{
		Network: network,
		Host:    host,
		Port:    port,
	}
}
