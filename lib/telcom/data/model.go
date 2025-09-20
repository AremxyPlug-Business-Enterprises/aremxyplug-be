package data

type DataInfo struct {
	UserID     string
	Network    int    `json:"network"`
	Plan       int    `json:"plan"`
	Mobile_Num string `json:"mobile_number"`
	Name       string `json:"name"`
	FullName   string
	ProviderID int
	PlanID     int
	Amount     string
	Plan_Name  string
	PlanSize   string
	Validity   string
}

type dontechAPIResponse struct {
	Id int `json:"id"`
	//Network       string `json:"network" bson:"network"`
	Plan_Name     string `json:"plan_name"`
	Plan_network  string `json:"plan_network"`
	Plan_amount   string `json:"plan_amount"`
	Mobile_number string `json:"mobile_number"`
	Ident         string `json:"ident"`
	Status        string `json:"Status"`
}

type easyaccessResponse struct {
	Status           string `json:"status"`
	Message          string `json:"message"`
	Reference        string `json:"reference"`
	Client_reference string `json:"client_reference"`
	Transaction_date string `json:"transaction_date"`
}

type api247Response struct {
	Status    string  `json:"status"`
	Message   string  `json:"message"`
	Response  string  `json:"response"`
	RequestID string  `json:"request-id"`
	Amount    float64 `json:"amount"`
	DataSize  string  `json:"data_size"`
	Network   string  `json:"network"`
	DataType  string  `json:"data_type"`
	OldWallet float64 `json:"old_wallet"`
	NewWallet float64 `json:"new_wallet"`
}
