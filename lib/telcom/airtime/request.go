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
