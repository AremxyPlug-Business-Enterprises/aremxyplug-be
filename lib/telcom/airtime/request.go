package airtime

type vendRequest struct {
	Amount        int    `json:"amount"`
	ProductCode   string `json:"product_code"`
	PhoneNumber   string `json:"phone_number"`
	Action        string `json:"action"`
	UserReference string `json:"user_reference"`
	BypassNetwork string `json:"bypass_network"`
}
type vendResponse struct {
	ServerMessage string        `json:"server_message"`
	Status        bool          `json:"status"`
	ErrorCode     int           `json:"error_code"`
	Data          vendData      `json:"data"`
	DataResult    []interface{} `json:"data_result"`
	ErrorData     []interface{} `json:"error_data"`
	TextStatus    string        `json:"text_status"`
	Error         interface{}   `json:"error"` // Could be null or other types
}

type vendData struct {
	Amount        string `json:"amount"`
	RechargeID    int    `json:"recharge_id"`
	AmountCharged string `json:"amount_charged"`
	Quantity      int    `json:"quantity"`
	AfterBalance  string `json:"after_balance"`
	TrueResponse  string `json:"true_response"`
	TextStatus    string `json:"text_status"`
	BonusEarned   string `json:"bonus_earned"`
}

type AirtimeInfo struct {
	UserID            string
	Network           string `json:"network"`
	Amount            string `json:"amount"`
	Phone_no          string `json:"mobileno"`
	Recipient         string `json:"recipient,omitempty"`
	Discount_percent  string
	Discount_amount   string
	FullName          string
	Reference         string
	TXN               string
	Profit_Margin     string
	Provider_Discount string
	ProviderName      string
}

type vtpassAirtimeResponse struct {
	Code                string                `json:"code"`
	Content             vtpassAirtimeContent  `json:"content"`
	ResponseDescription string                `json:"response_description"`
	RequestID           string                `json:"requestId"`
	Amount              string                `json:"amount"`
	TransactionDate     string                `json:"transaction_date"`
}

type vtpassAirtimeContent struct {
	Transactions vtpassAirtimeTransaction `json:"transactions"`
}

type vtpassAirtimeTransaction struct {
	Status          string                   `json:"status"`
	ProductName     string                   `json:"product_name"`
	UniqueElement   string                   `json:"unique_element"`
	UnitPrice       string                   `json:"unit_price"`
	Quantity        int                      `json:"quantity"`
	Commission      float64                  `json:"commission"`
	TotalAmount     float64                  `json:"total_amount"`
	Amount          string                   `json:"amount"`
	Phone           string                   `json:"phone"`
	TransactionID   string                   `json:"transactionId"`
	Type            string                   `json:"type"`
	Email           string                   `json:"email"`
	CommissionBreak vtpassCommissionDetails  `json:"commission_details"`
}

type vtpassCommissionDetails struct {
	Amount          float64 `json:"amount"`
	Rate            string  `json:"rate"`
	RateType        string  `json:"rate_type"`
	ComputationType string  `json:"computation_type"`
}

type AirtimeApiResponse struct {
	Success_Response string  `json:"success"`
	Message          string  `json:"message"`
	Network          string  `json:"network"`
	Phone_no         string  `json:"mobileno"`
	Amount           int     `json:"airtimeamount"`
	Charged          float64 `json:"amountcharged"`
	Status           string  `json:"status"`
	Date             string  `json:"transaction_date"`
	Reference        string  `json:"reference_no"`
}
