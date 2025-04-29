package service

import "fmt"

type FilterBy string

func (fb FilterBy) String() string {
	return string(fb)
}

var (
	filterByCountry FilterBy = "country"
)

type Filter struct {
	By    FilterBy
	Query string
}

func (f Filter) IsValid() bool {
	if f.By != "" && f.Query != "" {
		return true
	}
	return false
}

func NewFilter(by, query string) (*Filter, error) {
	fBy := FilterBy(by)
	// more filters to add
	if query == "" {
		return nil, fmt.Errorf("empty filter")
	}
	if fBy == filterByCountry {
		return &Filter{
			By:    fBy,
			Query: query,
		}, nil
	}
	return nil, fmt.Errorf("filterBy parameter '%s' not supported", by)
}
