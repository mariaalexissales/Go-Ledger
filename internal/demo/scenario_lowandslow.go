package demo

import (
	"context"
	"fmt"
)

func init() {
	Register(Scenario{
		Meta: Meta{
			ID:               "low-and-slow",
			Name:             "Distributed low-and-slow",
			Summary:          "Many clients, each staying just under the per-IP limit.",
			Teaches:          "Per-IP rate limiting only sees one IP at a time. Spread the same volume across enough sources and it never triggers, no matter how large the total gets.",
			Expect:           "More total requests than the burst scenario, and zero blocks. Watch the distinct-IP count instead of the block count.",
			Tags:             []string{"rate limiting", "evasion"},
			EstimatedSeconds: 6,
		},
		Run: runLowAndSlow,
	})
}

func runLowAndSlow(ctx context.Context, c *Client) error {
	limit := c.Policy().Limit

	// Each IP stops one short of the limit, so no single source ever trips it.
	perIP := min(limit-1, 6)
	if perIP < 1 {
		perIP = 1
	}

	hosts := min(maxRequestsPerRun/perIP, 20)

	c.Note(fmt.Sprintf("%d clients, %d requests each (%d total). The limit is %d per IP, and every client stays under it.",
		hosts, perIP, hosts*perIP, limit))

	for i := range perIP {
		for h := range hosts {
			ip := fmt.Sprintf("203.0.113.%d", 100+h)
			if _, err := c.Get(ctx, As(ip), fmt.Sprintf("/api/accounts/%d", (i+h)%20+1)); err != nil {
				return err
			}
		}
	}

	return nil
}
