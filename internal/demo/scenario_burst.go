package demo

import (
	"context"
	"fmt"
)

func init() {
	Register(Scenario{
		Meta: Meta{
			ID:               "burst",
			Name:             "Burst from one IP",
			Summary:          "A single client hammering the API as fast as it can.",
			Teaches:          "The case the rate limiter is built for. One identity, too many requests, refused for the rest of the block period.",
			Expect:           "The first requests succeed, then everything flips to 429 with a Retry-After countdown.",
			Tags:             []string{"rate limiting"},
			EstimatedSeconds: 3,
		},
		Run: runBurst,
	})
}

func runBurst(ctx context.Context, c *Client) error {
	const ip = "203.0.113.7"

	// Scaled to the live policy so the scenario still lands if the limit was
	// tuned in the console.
	limit := c.Policy().Limit
	total := min(limit*2+5, maxRequestsPerRun)

	c.Note(fmt.Sprintf("One IP, %d requests back to back. The limit is %d per %s.",
		total, limit, c.Policy().WindowText))

	for i := range total {
		if _, err := c.Get(ctx, As(ip), fmt.Sprintf("/api/accounts/%d", (i%20)+1)); err != nil {
			return err
		}
	}

	return nil
}
