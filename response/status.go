package xtremeres

type ResponseSuccess struct {
	Status Status `json:"status"`
	Result any    `json:"result"`
}

type ResponseSuccessWithPagination struct {
	Status     Status `json:"status"`
	Result     any    `json:"result"`
	Pagination any    `json:"pagination"`
}

type ResponseError struct {
	Status StatusError `json:"status"`
}

type StatusError struct {
	Status
	Bug bool `json:"bug"`
}

type Status struct {
	Code        int    `json:"code"`
	Message     string `json:"message"`
	InternalMsg string `json:"internalMsg"`
	Attributes  any    `json:"attributes"`
}
