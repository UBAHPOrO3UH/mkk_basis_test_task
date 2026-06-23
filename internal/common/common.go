package common

type APIResponse struct {
	Err    *string     `json:"err"`
	Result interface{} `json:"result"`
}

func SuccessResponse() APIResponse {
	return APIResponse{
		Result: "success",
	}
}

func ResultResponseNoErr(value interface{}) APIResponse {
	return APIResponse{
		Result: value,
		Err:    nil,
	}
}

func ResultResponseWithErr(value interface{}, err error) APIResponse {
	if err != nil {
		errMsg := err.Error()
		return APIResponse{
			Result: value,
			Err:    &errMsg,
		}
	}
	return ResultResponseNoErr(value)
}

func ErrorResponse(e error) APIResponse {
	errMsg := e.Error()
	return APIResponse{
		Err: &errMsg,
	}
}
