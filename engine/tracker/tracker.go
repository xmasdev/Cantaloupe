package tracker

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/xmasdev/Cantaloupe/engine/types"
)

func Announce(
	trackerURL string,
	req types.AnnounceRequest,
) (*types.AnnounceResponse, error) {
	parsed, err := url.Parse(trackerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid tracker URL: %w", err)
	}
	if strings.EqualFold(parsed.Scheme, "udp") {
		return announceUDP(trackerURL, req)
	}
	if !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https") {
		return nil, fmt.Errorf("unsupported tracker scheme %q", parsed.Scheme)
	}

	requestURL, err := buildAnnounceURL(trackerURL, req)
	if err != nil {
		return nil, err
	}

	body, err := requestTracker(requestURL)
	if err != nil {
		return nil, err
	}

	return parseAnnounceResponse(body)
}

// AnnounceMany tries tracker tiers in order and returns the first successful
// response. This supports torrents that publish several HTTP/HTTPS/UDP URLs.
func AnnounceMany(trackers []string, req types.AnnounceRequest) (*types.AnnounceResponse, error) {
	var lastErr error
	seen := make(map[string]struct{})
	for _, trackerURL := range trackers {
		trackerURL = strings.TrimSpace(trackerURL)
		if trackerURL == "" {
			continue
		}
		if _, exists := seen[trackerURL]; exists {
			continue
		}
		seen[trackerURL] = struct{}{}
		response, err := Announce(trackerURL, req)
		if err == nil {
			return response, nil
		}
		lastErr = fmt.Errorf("%s: %w", trackerURL, err)
	}
	if lastErr == nil {
		return nil, fmt.Errorf("torrent has no trackers")
	}
	return nil, lastErr
}
