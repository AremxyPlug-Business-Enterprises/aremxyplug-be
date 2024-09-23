package deposit

type paymentResponse struct {
	Data []paymentData `json:"data"`
}

type paymentData struct {
	ID            string               `json:"id"`
	Type          string               `json:"type"`
	Attributes    paymentAttributes    `json:"attributes"`
	Relationships paymentRelationships `json:"relationships"`
}

type paymentAttributes struct {
	CreatedAt        string       `json:"createdAt"`
	Amount           float64      `json:"amount"`
	PaymentReference string       `json:"paymentReference"`
	Fee              int          `json:"fee"`
	Narration        string       `json:"narration"`
	PaidAt           string       `json:"paidAt"`
	CounterParty     counterParty `json:"counterParty"`
	Currency         string       `json:"currency"`
	Type             string       `json:"type"`
}

type counterParty struct {
	AccountNumber string `json:"accountNumber"`
	AccountName   string `json:"accountName"`
	Bank          bank   `json:"bank"`
}

type bank struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	CbnCode string `json:"cbnCode"`
	NipCode string `json:"nipCode"`
}

type paymentRelationships struct {
	SettlementAccount relationship `json:"settlementAccount"`
	VirtualNuban      relationship `json:"virtualNuban"`
	Settlements       settlements  `json:"settlements"`
}

type relationship struct {
	Data relationshipData `json:"data"`
}

type relationshipData struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type settlements struct {
	Data []relationshipData `json:"data"`
}
