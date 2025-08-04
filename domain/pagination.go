package domain

type Pagination struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalPages int   `json:"total_pages"`
	TotalItems int64 `json:"total_items"`
}

func NewPagination(page, perPage, totalPages int, totalItems int64) Pagination {
	return Pagination{
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		TotalItems: totalItems,
	}
}
