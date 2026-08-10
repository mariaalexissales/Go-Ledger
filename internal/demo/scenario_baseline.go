package demo

import (
	"context"
	"fmt"
	"time"
)

func init() {
	Register(Scenario{
		Meta: Meta{
			ID:      "baseline",
			Name:    "Normal traffic",
			Summary: "A handful of ordinary clients browsing the ledger at a human pace.",
			Teaches: "What a healthy log looks like. Every scenario after this one should be read against it.",
			Expect:  "Every request ALLOWED. No blocks, no gaps.",
			Tags:    []string{"baseline"},
			// Four IPs, three requests each, ~250ms apart.
			EstimatedSeconds: 4,
		},
		Run: runBaseline,
	})
}

func runBaseline(ctx context.Context, c *Client) error {
	ips := []string{"198.51.100.11", "198.51.100.12", "198.51.100.13", "198.51.100.14"}

	c.Note("Four clients, three requests each, spaced like real browsing.")

	for round := 1; round <= 3; round++ {
		for i, ip := range ips {
			if _, err := c.Get(ctx, As(ip), fmt.Sprintf("/api/accounts/%d", i+1)); err != nil {
				return err
			}
		}

		if err := c.Sleep(ctx, 250*time.Millisecond); err != nil {
			return err
		}
	}

	return nil
}
