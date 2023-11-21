package models

type TransferInfo struct {
	Bank_name      string `json:"bank_name"`
	Account_Number string `json:"account_number"`
	Account_Name   string `json:"account_name"`
	Amount         string `json:"amount"`
	Reason         string `json:"message"`
}

type TransferResponse struct {
	Bank_Name      string `json:"bank_name"`
	Account_Name   string `json:"account_name"`
	Account_No     string `json:"account_no"`
	Name           string `json:"name"`
	Product        string `json:"product"`
	Description    string `json:"description"`
	Reason         string `json:"message"`
	Order_ID       int    `json:"order_id"`
	Transaction_ID string `json:"transaction_id"`
	Session_ID     string `json:"session_id"`
}

type AccountDetails struct {
	Bank_Name    string `json:"bank_name"`
	Account_Name string `json:"account_name"`
	Account_No   string `json:"account_no"`
}

type CounterParty struct {
	ID            string `json:"id"`
	AccountName   string `json:"account_name"`
	AccountNumber string `json:"account_number"`
	BankName      string `json:"bank_name"`
	NIPCode       string `json:"nipCode"`
}
type BankDetails struct {
	Name    string `json:"name"`
	NIPCode string `json:"nipCode"`
}
