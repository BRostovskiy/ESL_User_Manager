package common

/*
	Global variables are evil, but sometimes it's easiest solution
*/

var AllowedFilters Filters

func init() {
	AllowedFilters = Filters{}
}

type Filters map[string]struct{}

func (af Filters) WithValues(filters []string) Filters {
	for _, v := range filters {
		if _, ok := af[v]; !ok {
			af[v] = struct{}{}
		}
	}
	return af
}
