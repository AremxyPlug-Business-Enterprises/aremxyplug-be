package tvsub

type serverResponse struct {
	Code    string  `json:"code"`
	Content content `json:"content"`
}

type content struct {
	CustomerName      string            `json:"Customer_Name"`
	Status            string            `json:"Status"`
	DueDate           string            `json:"Due_Date"`
	CustomerNumber    string            `json:"Customer_Number"`
	CustomerType      string            `json:"Customer_Type"`
	CommissionDetails commissionDetails `json:"commission_details"`
}

type commissionDetails struct {
	Amount          *float64 `json:"amount"` // Use pointer to handle null values
	Rate            string   `json:"rate"`
	RateType        string   `json:"rate_type"`
	ComputationType string   `json:"computation_type"`
}

type verifyResponse struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}
