package telcom

type TelcomRecipient struct {
	UserID    string      `json:"user_id" bson:"user_id"`
	Recipient []Recipient `json:"recipients" bson:"recipients"`
}

type Recipient struct {
	ID       int    `json:"id" bson:"id"`
	Network  string `json:"network" bson:"network"`
	Phone_no string `json:"phone" bson:"phone"`
	Name     string `json:"name,omitempty" bson:"name"`
}
