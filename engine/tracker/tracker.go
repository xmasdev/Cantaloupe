package tracker

import "github.com/xmasdev/Cantaloupe/engine/types"

func Announce(
	trackerURL string,
	req types.AnnounceRequest,
) (*types.AnnounceResponse, error) {

	url, err := buildAnnounceURL(trackerURL, req)
	if err != nil {
		return nil, err
	}

	body, err := requestTracker(url)
	if err != nil {
		return nil, err
	}

	return parseAnnounceResponse(body)
}
