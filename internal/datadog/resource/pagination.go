package resource

import "fmt"

// PaginationParams holds pagination state for API requests.
type PaginationParams struct {
	// For offset-based pagination (dashboards)
	Start int
	Count int

	// For page-based pagination (monitors)
	Page     int
	PageSize int
}

// NewOffsetPagination creates pagination params for offset-based APIs (start/count).
func NewOffsetPagination(pageSize int) *PaginationParams {
	return &PaginationParams{
		Start: 0,
		Count: pageSize,
	}
}

// NewPagePagination creates pagination params for page-based APIs (page/page_size).
func NewPagePagination(pageSize int) *PaginationParams {
	return &PaginationParams{
		Page:     0,
		PageSize: pageSize,
	}
}

// NextOffsetPage advances to the next page using offset-based pagination.
// Returns true if there might be more pages (based on items received).
func (p *PaginationParams) NextOffsetPage(itemsReceived int) bool {
	if itemsReceived == 0 || itemsReceived < p.Count {
		return false
	}
	p.Start += itemsReceived
	return true
}

// NextPage advances to the next page using page-based pagination.
// Returns true if there might be more pages (based on items received).
func (p *PaginationParams) NextPage(itemsReceived int) bool {
	if itemsReceived == 0 || itemsReceived < p.PageSize {
		return false
	}
	p.Page++
	return true
}

// FormatOffsetURL formats a URL with start/count pagination parameters.
func (p *PaginationParams) FormatOffsetURL(baseURL string) string {
	return fmt.Sprintf("%s?start=%d&count=%d", baseURL, p.Start, p.Count)
}

// FormatPageURL formats a URL with page/page_size pagination parameters.
func (p *PaginationParams) FormatPageURL(baseURL string) string {
	return fmt.Sprintf("%s?page=%d&page_size=%d", baseURL, p.Page, p.PageSize)
}
