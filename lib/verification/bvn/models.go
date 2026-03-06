package bvn

type bvnRequest struct {
	BVN_number string `json:"number"`
}

type bvnResponse struct {
	Status       bool      `json:"status"`
	Detail       string    `json:"detail"`
	ResponseCode string    `json:"response_code"`
	Data         *userData `json:"data"`
}

type userData struct {
	FirstName   string `json:"firstName"`
	MiddleName  string `json:"middleName"`
	LastName    string `json:"lastName"`
	DateOfBirth string `json:"dateOfBirth"`
	PhoneNumber string `json:"phoneNumber"`
}

type BVNVerificationResult struct {
	Success         bool   `json:"success"`          // Overall verification status
	NameMatched     bool   `json:"name_matched"`     // Whether names match
	DOB             string `json:"dob"`              // Date of birth (if available)
	ResponseCode    string `json:"response_code"`    // API response code
	ResponseMessage string `json:"response_message"` // API response message
	Reason          string `json:"reason,omitempty"` // Reason for failure (if any)
}
