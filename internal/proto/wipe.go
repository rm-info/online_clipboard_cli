package proto

import (
	"context"
	"fmt"
	"net/http"
)

// Wipe deletes the session immediately. The server replies with a 303
// redirect to /; since we don't follow redirects, we accept any 2xx/3xx
// status as success.
func (c *Client) Wipe(ctx context.Context, sid, cookie string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/"+sid+"/wipe"), nil)
	if err != nil {
		return err
	}
	setCookie(req, sid, cookie)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST /%s/wipe: %w", sid, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return nil
	}
	return parseHTTPError(resp)
}
