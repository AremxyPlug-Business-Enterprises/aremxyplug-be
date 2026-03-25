package nin

type ninRequest struct {
	NIN_number string `json:"number"`
}

type ninResponse struct {
	Status       bool    `json:"status"`
	Detail       string  `json:"detail"`
	ResponseCode string  `json:"response_code"`
	NINData      ninData `json:"nin_data"`
}

// NINData represents the nested "nin_data" object
type ninData struct {
	FirstName   string `json:"firstname"`
	Gender      string `json:"gender"`
	Surname     string `json:"surname"`
	MiddleName  string `json:"middlename"`
	Birthdate   string `json:"birthdate"`
	TelephoneNo string `json:"telephoneno"`
}

type NINVerificationResult struct {
	Success         bool   `json:"success"`          // Overall verification status
	NameMatched     bool   `json:"name_matched"`     // Whether names match
	Birthdate       string `json:"birthdate"`        // Birthdate (if available)
	ResponseCode    string `json:"response_code"`    // API response code
	ResponseMessage string `json:"response_message"` // API response message
	Reason          string `json:"reason,omitempty"` // Reason for failure (if any)
}
