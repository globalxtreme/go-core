package xtrememodel

import (
	"gorm.io/gorm"
	"net/url"
	"strconv"
)

type Pagination struct {
	Count        int
	CurrentPage  any
	LinkPrevious int
	LinkNext     int
	PerPage      int
	TotalPage    int
}

func (p Pagination) ParsePagination() map[string]interface{} {
	return map[string]interface{}{
		"count":       p.Count,
		"currentPage": p.CurrentPage,
		"perPage":     p.PerPage,
		"totalPage":   p.TotalPage,
		"links": map[string]interface{}{
			"next":     p.LinkNext,
			"previous": p.LinkPrevious,
		},
	}
}

func Paginate(parameters url.Values, query *gorm.DB, model interface{}) (*gorm.DB, Pagination) {
	var count int64
	query.Model(&model).Count(&count)

	page, _ := strconv.Atoi(parameters.Get("page"))
	if page == 0 {
		page = 1
	}

	limit, _ := strconv.Atoi(parameters.Get("limit"))
	if limit == 0 {
		limit = 50
	}

	offset := (page - 1) * limit
	query = query.Limit(limit).Offset(offset)

	dataPagination := Pagination{
		Count:       int(count),
		PerPage:     limit,
		CurrentPage: page,
		TotalPage:   (int(count) / limit) + 1,
	}

	if page < dataPagination.TotalPage {
		dataPagination.LinkNext = page + 1
	}

	if page > 1 {
		dataPagination.LinkPrevious = page - 1
	}

	return query, dataPagination
}
