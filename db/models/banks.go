package models

type TransferInfo struct {
	Bank_name      string `json:"bank_name"`
	Account_Number string `json:"account_number"`
	Amount         string `json:"amount"`
	Reason         string `json:"message"`
}

type BankApiResponse struct {
}

type TransferResponse struct {
	Bank_Name      string `json:"bank_name"`
	Account_Name   string `json:"account_name"`
	Account_No     string `json:"account_no"`
	Name           string `json:"name"`
	Product        string `json:"product"`
	Description    string `json:"description"`
	Reason         string `json:"message"`
	Order_ID       string `json:"order_id"`
	Transaction_ID string `json:"transaction_id"`
	Session_ID     string `json:"session_id"`
}

type AccountDetails struct {
	Bank_Name    string `json:"bank_name"`
	Account_Name string `json:"account_name"`
	Account_No   string `json:"account_no"`
}
