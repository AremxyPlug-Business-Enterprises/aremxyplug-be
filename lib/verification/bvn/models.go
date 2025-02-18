package bvn

type bvnRequest struct {
	BVN_number string `json:"number"`
}

type bvnResponse struct {
	Status       bool        `json:"status"`
	Detail       string      `json:"detail"`
	ResponseCode string      `json:"response_code"`
	Data         *userData   `json:"data"`
	Source       string      `json:"source"`
	UserInfo     interface{} `json:"user_info"` // Can be changed to a specific type if needed
	RequestData  requestData `json:"request_data"`
}

type userData struct {
	FirstName   string `json:"firstName"`
	MiddleName  string `json:"middleName"`
	LastName    string `json:"lastName"`
	DateOfBirth string `json:"dateOfBirth"`
	PhoneNumber string `json:"phoneNumber"`
}

type requestData struct {
	Number string `json:"number"`
}

type BVNVerificationResult struct {
	Success         bool   `json:"success"`          // Overall verification status
	NameMatched     bool   `json:"name_matched"`     // Whether names match
	ResponseCode    string `json:"response_code"`    // API response code
	ResponseMessage string `json:"response_message"` // API response message
	Reason          string `json:"reason,omitempty"` // Reason for failure (if any)
}
