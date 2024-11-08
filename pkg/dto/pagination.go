package dto

import (
	"fmt"
	"strings"
)

const defaultLimit = 20
const defaultPage = 1
const defaultSort = "created_date desc"

type Pagination struct {
	Page     *int    `json:"page" form:"page" url:"page"`
	PageSize *int    `json:"page_size" form:"page_size" url:"page_size"`
	Sort     *string `json:"sort" form:"sort" url:"sort"`
}

func (p Pagination) StartOffset() int {
	page := defaultPage
	pageSize := defaultLimit
	if p.Page != nil {
		page = *p.Page
	}

	if p.PageSize != nil {
		pageSize = *p.PageSize
	}
	page = page - 1

	start := page * pageSize
	return start
}

func (p Pagination) Limit() int {
	if p.PageSize == nil {
		return defaultLimit
	}
	return *p.PageSize
}

func (p Pagination) CurrentPage() int {
	if p.Page != nil {
		return *p.Page
	}
	return defaultPage
}

func (p Pagination) SortOrderText() string {
	sort := defaultSort
	if p.Sort != nil {
		splitted := strings.Split(*p.Sort, "-")
		if len(splitted) == 2 {
			sort = fmt.Sprintf("%s %s", splitted[0], splitted[1])
		}
	}
	return sort
}
