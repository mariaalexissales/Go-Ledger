package demo

import (
	"context"
	"fmt"
)

func init() {
	Register(Scenario{
		Meta: Meta{
			ID:               "enumeration",
			Name:             "Account enumeration",
			Summary:          "Walking account IDs in order, including ones that do not exist.",
			Teaches:          "Rate limiting catches the volume, but the log is what shows the intent: one IP, sequential IDs, a trail of 404s. That shape is worth alerting on even when nothing is blocked.",
			Expect:           "A run of 200s and 404s in ID order, then 429s once the limit is reached. Note how obvious the pattern is in the event feed.",
			Tags:             []string{"reconnaissance"},
			EstimatedSeconds: 3,
		},
		Run: runEnumeration,
	})
}

func runEnumeration(ctx context.Context, c *Client) error {
	const ip = "203.0.113.66"

	scanTo := min(c.Policy().Limit+20, maxRequestsPerRun)

	c.Note(fmt.Sprintf("Scanning /api/accounts/1 through /api/accounts/%d from a single IP.", scanTo))

	for id := 1; id <= scanTo; id++ {
		if _, err := c.Get(ctx, As(ip), fmt.Sprintf("/api/accounts/%d", id)); err != nil {
			return err
		}
	}

	return nil
}
