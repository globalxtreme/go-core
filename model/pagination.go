package xtrememodel

import (
	"gorm.io/gorm"
	"net/url"
	"strconv"
)

type Pagination struct {
	Total        int
	Count        int
	CurrentPage  any
	LinkPrevious int
	LinkNext     int
	PerPage      int
	TotalPage    int
}

func (p Pagination) ParsePagination() map[string]interface{} {
	return map[string]interface{}{
		"total":       p.Total,
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

func Paginate[M any](query *gorm.DB, parameters url.Values, model M) ([]M, interface{}, error) {
	var data []M

	var total int64
	query.Model(&model).Count(&total)

	page, _ := strconv.Atoi(parameters.Get("page"))
	if page == 0 {
		page = 1
	}

	limit, _ := strconv.Atoi(parameters.Get("limit"))
	if limit == 0 {
		limit = 50
	}

	offset := (page - 1) * limit
	res := query.Limit(limit).Offset(offset).Find(&data)
	if res.Error != nil {
		return nil, nil, res.Error
	}

	dataPagination := Pagination{
		Total:       int(total),
		Count:       int(res.RowsAffected),
		PerPage:     limit,
		CurrentPage: page,
		TotalPage:   (int(total) / limit) + 1,
	}

	if page < dataPagination.TotalPage {
		dataPagination.LinkNext = page + 1
	}

	if page > 1 {
		dataPagination.LinkPrevious = page - 1
	}

	return data, dataPagination.ParsePagination(), nil
}
