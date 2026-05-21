package proto

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Challenge is a single-use proof-of-work issue from /pow/challenge.
type Challenge struct {
	Challenge  string `json:"challenge"`
	Difficulty int    `json:"difficulty"`
}

// FetchChallenge requests a fresh PoW challenge. Each call returns a
// new (challenge, difficulty) pair; the server invalidates a challenge
// the moment it's consumed (or after POW_CHALLENGE_TTL_SECONDS).
func (c *Client) FetchChallenge(ctx context.Context) (*Challenge, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url("/pow/challenge"), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET /pow/challenge: %w", err)
	}
	defer resp.Body.Close()
	if err := standardStatusErrors(resp); err != nil {
		return nil, err
	}
	ch := &Challenge{}
	if err := json.NewDecoder(resp.Body).Decode(ch); err != nil {
		return nil, fmt.Errorf("decode challenge: %w", err)
	}
	return ch, nil
}
