package electricity

type serverResponse struct {
	Code    string  `json:"code"`
	Content content `json:"content"`
}

type content struct {
	Name         string `json:"name"`
	Meter_Number string `json:"meter_number"`
	Err          string `json:"error,omitempty"`
}

type verifyResponse struct {
	Name     string `json:"name"`
	Meter_No string `json:"meter_number"`
}
