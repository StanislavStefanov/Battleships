package utils

type Request struct {
	PlayerId string                 `json:"playerId"`
	Action   string                 `json:"action"`
	Args     map[string]interface{} `json:"args"`
}

func BuildRequest(id string, action string, args map[string]interface{}) Request {
	return Request{
		PlayerId: id,
		Action:   action,
		Args:     args,
	}
}

func (r *Request) GetId() string {
	return r.PlayerId
}

func (r *Request) GetAction() string {
	return r.Action
}

func (r *Request) GetArgs() map[string]interface{} {
	return r.Args
}

type Response struct {
	Action  string                 `json:"action"`
	Message string                 `json:"message"`
	Args    map[string]interface{} `json:"args"`
}

func BuildResponse(action string, message string, args map[string]interface{}) Response {
	return Response{
		Action:  action,
		Message: message,
		Args:    args,
	}
}

func (r *Response) GetAction() string {
	return r.Action
}

func (r *Response) GetMessage() string {
	return r.Message
}

func (r *Response) GetArgs() map[string]interface{} {
	return r.Args
}
