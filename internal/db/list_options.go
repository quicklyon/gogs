package db

import "gogs.io/gogs/internal/conf"

// ListOptions options to paginate results
type ListOptions struct {
	PageSize int
	Page     int // start from 1
}

// GetStartEnd returns the start and end of the ListOptions
func (opts *ListOptions) GetStartEnd() (start, end int) {
	opts.setDefaultValues()
	start = (opts.Page - 1) * opts.PageSize
	end = start + opts.PageSize
	return
}

func (opts *ListOptions) setDefaultValues() {
	if opts.PageSize <= 0 {
		opts.PageSize = 30
	}
	if opts.PageSize > conf.API.MaxResponseItems {
		opts.PageSize = conf.API.MaxResponseItems
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}
}
