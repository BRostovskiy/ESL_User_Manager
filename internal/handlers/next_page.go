package handlers

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/BorisRostovskiy/ESL/internal/service"
)

const (
	milliseconds30Minutes = 1800000
)

type NextPage struct {
	Limit    int       `json:"limit"`
	Offset   int       `json:"offset"`
	FilterBy string    `json:"filter_by"`
	Filter   string    `json:"filter"`
	Time     time.Time `json:"time"`
}

func GenerateNextPage(limit, offset, numUsers int, filter *service.Filter) (string, error) {
	if (limit > 0 && offset >= 0) && (numUsers-(offset+limit) > 0) {
		nextPage := NextPage{
			Offset:   limit,
			Limit:    limit,
			Time:     time.Now(),
			Filter:   "",
			FilterBy: "",
		}
		if filter != nil && filter.IsValid() {
			nextPage.FilterBy = filter.By.String()
			nextPage.Filter = filter.Query
		}
		var np []byte
		np, err := json.Marshal(nextPage)

		if err != nil {
			return "", err
		}
		return b64.StdEncoding.EncodeToString(np), nil
	}
	return "", nil
}

// LoadNextPage helper function to load and validate next page criteria
func LoadNextPage(nPage, filter, filterBy string, paginationFn func() (int, error)) (*NextPage, error) {
	np := &NextPage{
		Limit: -1,
	}

	// ignore other parameters if next_page is loaded, and it was made less than 30 minutes ago
	if nPage != "" {
		sDec, err := b64.StdEncoding.DecodeString(nPage)
		if err != nil {
			return nil, fmt.Errorf("could not decode next_page argument: %w", err)
		}

		if err = json.Unmarshal(sDec, np); err != nil {
			return nil, fmt.Errorf("could not unmarshal limit offset: %w", err)
		}

		// if next_page were build more than 30 minutes ago, let's consider it as an expired one
		if time.Since(np.Time).Milliseconds() < milliseconds30Minutes {
			return np, nil
		}
	}

	// if we are here, then either next page was not used, or it was expired.
	np.Limit = -1
	np.Offset = 0

	pagination, err := paginationFn()
	if err != nil {
		return nil, err
	}

	if pagination > 0 && pagination != np.Limit {
		np.Limit = pagination
	}
	if (filter == "" && filterBy != "") || (filter != "" && filterBy == "") {
		return nil, fmt.Errorf("parameters filter and filterBy should be used together")
	}
	if filter != "" && filterBy != "" {
		np.Filter = filter
		np.FilterBy = filterBy
	}

	return np, nil
}
