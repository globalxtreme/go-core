package response

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Array      []interface{}
	Object     interface{}
	Pagination *any
}

func (res Response) Success(w http.ResponseWriter, args ...string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	result := res.Object
	if len(res.Array) > 0 {
		result = res.Array
	}

	internalMsg := ""
	if len(args) > 0 {
		internalMsg = args[0]
	}

	status := Status{
		Code:        http.StatusOK,
		Message:     "Success",
		InternalMsg: internalMsg,
		Attributes:  nil,
	}

	if res.Pagination != nil {
		json.NewEncoder(w).Encode(ResponseSuccessWithPagination{
			Status:     status,
			Result:     result,
			Pagination: res.Pagination,
		})
	} else {
		json.NewEncoder(w).Encode(ResponseSuccess{
			Status: status,
			Result: result,
		})
	}
}
