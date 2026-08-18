package tracker

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func requestTracker(trackerURL string) ([]byte, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(trackerURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"tracker returned HTTP status %s",
			resp.Status,
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
