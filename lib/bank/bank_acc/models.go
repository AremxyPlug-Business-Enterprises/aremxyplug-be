package bankacc

type bank struct {
	Name    string `json:"name"`
	NipCode string `json:"nipCode"`
}

// Attributes represents the attributes in the JSON.
type virtualAttributes struct {
	Bank          bank   `json:"bank"`
	AccountName   string `json:"accountName"`
	Permanent     bool   `json:"permanent"`
	Currency      string `json:"currency"`
	AccountNumber string `json:"accountNumber"`
	Status        string `json:"status"`
}

// Data represents the data in the JSON.
type data struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Attributes virtualAttributes `json:"attributes"`
}
type virtualNubanAttributes struct {
	VirtualAccount virtualAccountDetail `json:"virtualAccountDetail"`
	Provider       string               `json:"provider"`
}

type virtualAccountDetail struct {
	Name      string `json:"name"`
	BVN       string `json:"bvn"`
	Email     string `json:"email"`
	Permanent bool   `json:"permanent"`
}

type accountData struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type virtualAccountRelationships struct {
	SettlementAccount struct {
		Data accountData `json:"data"`
	} `json:"settlementAccount"`
}

type virtualNubanData struct {
	Type          string                      `json:"type"`
	Attributes    virtualNubanAttributes      `json:"attributes"`
	Relationships virtualAccountRelationships `json:"relationships"`
}

type virtualNubanPayload struct {
	Data virtualNubanData `json:"data"`
}

type virtualAccountResponse struct {
	Data data `json:"data"`
}
