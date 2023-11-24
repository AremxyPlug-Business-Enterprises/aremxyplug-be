package transfer

type intiateTransfer struct {
	Data          transferData  `json:"data"`
	Type          string        `json:"type"`
	Account       account       `json:"account"`
	Relationships relationships `json:"relationships"`
	CounterParty  counterParty  `json:"counterParty"`
}

type transferData struct {
	Attributes transferDataAttributes `json:"attributes"`
}

type transferDataAttributes struct {
	Currency  string  `json:"currency"`
	Amount    float64 `json:"amount"`
	Reason    string  `json:"reason,omitempty"`
	Reference string  `json:"reference,omitempty"`
}

type account struct {
	Data data `json:"data"`
}

type relationships struct {
	DestinationAcc destination `json:"destinationAccount"`
}

type destination struct {
	Data struct {
		Type string `json:"type"`
	} `json:"data"`
}

type counterParty struct {
	Data data `json:"data"`
}

type data struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type transferResult struct {
	Data transferResultData `json:"data"`
}

type transferResultData struct {
	Type       string                   `json:"type"`
	Attributes transferResultAttributes `json:"attributes"`
}

type transferResultAttributes struct {
	Reason  string  `json:"reason"`
	Ammount float64 `json:"ammount"`
	Status  string  `json:"status"`
}

type counterPartyAPIResponse struct {
	Data counterpartyResponseData `json:"data"`
}

type counterpartyResponseData struct {
	Type         string `json:"type"`
	ID           string `json:"id"`
	AccountName  string `json:"accountName"`
	AccontNumber string `json:"accontNumber"`
	Status       string `json:"status"`
	Bank         bank   `json:"bank"`
}

type bank struct {
	Name    string `json:"name"`
	NipCode string `json:"nipCode"`
}

type verifyAccountResponse struct {
	Data verifyAccountData `json:"data"`
}

type verifyAccountData struct {
	AccountName   string                  `json:"accountName"`
	AccountNumber string                  `json:"accountNumber"`
	Attributes    verifyAccountAttributes `json:"attributes"`
}

type verifyAccountAttributes struct {
	Bank bank `json:"bank"`
}

type counterPartyPayload struct {
	Data counterpartyData `json:"data"`
}

type counterpartyData struct {
	Type          string                   `json:"type"`
	Attributes    counterpartyAttributes   `json:"attributes"`
	Relationships counterpartyRelationship `json:"relationships"`
}

type counterpartyAttributes struct {
	VerifyName    bool   `json:"verifyName"`
	AccountName   string `json:"accountName"`
	AccountNumber string `json:"accountNumber"`
	BankCode      string `json:"bankCode"`
}

type counterpartyRelationship struct {
	Bank struct {
		Data accountData `json:"data"`
	} `json:"bank"`
}

type accountData struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type bankLists struct {
	BanksData []bankData `json:"data"`
}

type bankAttributes struct {
	NIPCode string `json:"nipCode"`
	Name    string `json:"name"`
}

type bankData struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Atrributes bankAttributes `json:"attributes"`
}
