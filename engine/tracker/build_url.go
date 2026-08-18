package tracker

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/xmasdev/Cantaloupe/engine/types"
)

func buildAnnounceURL(trackerURL string, req types.AnnounceRequest) (string, error) {
	u, err := url.Parse(trackerURL)
	if err != nil {
		return "", fmt.Errorf("invalid tracker URL: %w", err)
	}

	params := []string{
		"info_hash=" + percentEncodeBytes(req.InfoHash[:]),
		"peer_id=" + percentEncodeBytes(req.PeerID[:]),
		"port=" + strconv.Itoa(int(req.Port)),
		"uploaded=" + strconv.FormatInt(req.Uploaded, 10),
		"downloaded=" + strconv.FormatInt(req.Downloaded, 10),
		"left=" + strconv.FormatInt(req.Left, 10),
		"compact=1",
	}

	if req.Event != "" {
		params = append(params, "event="+url.QueryEscape(req.Event))
	}

	u.RawQuery = strings.Join(params, "&")

	return u.String(), nil
}

func percentEncodeBytes(data []byte) string {
	const hex = "0123456789ABCDEF"

	var b strings.Builder

	for _, c := range data {
		// RFC 3986 unreserved characters.
		if (c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '-' ||
			c == '.' ||
			c == '_' ||
			c == '~' {
			b.WriteByte(c)
			continue
		}

		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&0x0F])
	}

	return b.String()
}
