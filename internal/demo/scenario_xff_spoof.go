package demo

import (
	"context"
	"fmt"
)

func init() {
	Register(Scenario{
		Meta: Meta{
			ID:                     "xff-spoof",
			Name:                   "X-Forwarded-For spoofing",
			Summary:                "One machine sending a different forged X-Forwarded-For on every request.",
			Teaches:                "The guard reads X-Forwarded-For verbatim, and any client can set that header. A single attacker gets a fresh identity per request, so a per-IP limit never accumulates. The fix is to trust forwarding headers only from a known proxy. Switch client IP mode to remote-addr and run this again.",
			Expect:                 "In xff-trust-all mode: far past the limit, zero blocks, and a log full of source IPs that do not exist. In remote-addr mode: blocked almost immediately.",
			Tags:                   []string{"header spoofing", "evasion"},
			EstimatedSeconds:       4,
			RequiresVulnerableMode: true,
		},
		Run: runXFFSpoof,
	})
}

func runXFFSpoof(ctx context.Context, c *Client) error {
	limit := c.Policy().Limit
	total := min(limit*3, maxRequestsPerRun)

	c.Note(fmt.Sprintf("One client, %d requests, a new forged X-Forwarded-For each time. The limit is %d.", total, limit))

	for i := range total {
		// Spoof, not As: this scenario exists to test whether the server
		// believes an untrusted header.
		ip := fmt.Sprintf("198.18.%d.%d", i/256, i%256)
		if _, err := c.Get(ctx, Spoof(ip), fmt.Sprintf("/api/accounts/%d", (i%20)+1)); err != nil {
			return err
		}
	}

	return nil
}
