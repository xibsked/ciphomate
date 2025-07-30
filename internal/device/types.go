package device

type StatusResponse struct {
	Result []struct {
		Code  string      `json:"code"`
		Value interface{} `json:"value"`
	} `json:"result"`
}

type Command struct {
	Code  string      `json:"code"`
	Value interface{} `json:"value"`
}

type CommandRequest struct {
	Commands []Command `json:"commands"`
}
