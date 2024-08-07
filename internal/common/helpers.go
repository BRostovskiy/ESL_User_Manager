package common

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	pb "github.com/BorisRostovskiy/ESL/internal/servers/grpc/gen/user-manager"
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

func GenerateNextPage(limit, offset, numUsers int, filter, filterBy string) (string, error) {
	if (limit > 0 && offset >= 0) && (numUsers-(offset+limit) > 0) {
		var np []byte
		np, err := json.Marshal(NextPage{
			Offset:   limit,
			Limit:    limit,
			Time:     time.Now(),
			Filter:   filter,
			FilterBy: filterBy,
		})

		if err != nil {
			return "", err
		}
		return b64.StdEncoding.EncodeToString(np), nil
	}
	return "", nil
}

func NextPageFromHTTP(r *http.Request) (*NextPage, error) {
	return loadNextPage(r.URL.Query().Get("next_page"),
		r.URL.Query().Get("filter"),
		r.URL.Query().Get("filterBy"),
		func() (int, error) {
			if r.URL.Query().Get("pagination") != "" {
				p, err := strconv.ParseInt(r.URL.Query().Get("pagination"), 10, 64)
				if err != nil {
					return -1, fmt.Errorf("malformed pagination")
				}
				return int(p), nil
			}
			return -1, nil
		},
	)
}

func NextPageFromPB(r *pb.ListUsersRequest) (*NextPage, error) {
	return loadNextPage(r.NextPage, r.Filter, r.FilterBy, func() (int, error) {
		return int(r.Pagination), nil
	})
}

// loadNextPage helper function to load and validate next page criteria
func loadNextPage(nPage, filter, filterBy string, paginationFn func() (int, error)) (*NextPage, error) {
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
		if time.Now().Sub(np.Time).Milliseconds() < milliseconds30Minutes {
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
