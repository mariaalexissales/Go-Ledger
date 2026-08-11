package demo

import (
	"context"
	"fmt"
	"time"
)

// scenarios is every demo the server offers, in reading order. Adding one means
// adding an entry here; nothing else needs wiring.
//
// Note how each Run scales itself to c.Policy().Limit rather than hardcoding a
// request count. That is what keeps these meaningful after the limit is retuned
// in the console -- a scenario built for a limit of 30 would stop demonstrating
// anything at a limit of 5.
var scenarios = []Scenario{
	{
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
	},
	{
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
	},
	{
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
	},
	{
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
	},
	{
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
	},
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

func runLowAndSlow(ctx context.Context, c *Client) error {
	limit := c.Policy().Limit

	// Six requests per IP, or one under the limit if the limit is tighter than
	// that. At the default limit of 30 this is 6 -- well under, not just under,
	// because the point is the distributed total rather than how close each
	// source gets to tripping.
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
