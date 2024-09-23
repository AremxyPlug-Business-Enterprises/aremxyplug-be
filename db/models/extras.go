package models

type Referral struct {
	UserID  string `json:"user_id"`
	RefCode string `json:"ref_code"`
	Count   int    `json:"count"`
}

type Points struct {
	UserID  string `json:"user_id"`
	Balance int    `json:"balance"`
}
