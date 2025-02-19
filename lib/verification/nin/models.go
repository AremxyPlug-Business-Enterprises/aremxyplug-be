package nin

type ninRequest struct {
	NIN_number string `json:"number_nin"`
}

type ninResponse struct {
	Status       bool    `json:"status"`
	Detail       string  `json:"detail"`
	ResponseCode string  `json:"response_code"`
	NINData      ninData `json:"nin_data"`
}

// NINData represents the nested "nin_data" object
type ninData struct {
	BirthCountry     string `json:"birthcountry"`
	BirthDate        string `json:"birthdate"`
	BirthLGA         string `json:"birthlga"`
	BirthState       string `json:"birthstate"`
	CentralID        string `json:"centralID"`
	EducationalLevel string `json:"educationallevel"`
	Email            string `json:"email"`
	EmploymentStatus string `json:"employmentstatus"`
	FirstName        string `json:"firstname"`
	Gender           string `json:"gender"`
	Height           string `json:"heigth"`
	MaritalStatus    string `json:"maritalstatus"`
	MiddleName       string `json:"middlename"`
	NIN              string `json:"nin"`
	NOKAddress1      string `json:"nok_address1"`
	NOKAddress2      string `json:"nok_address2"`
	NOKFirstName     string `json:"nok_firstname"`
	NOKLGA           string `json:"nok_lga"`
	NOKMiddleName    string `json:"nok_middlename"`
	NOKPostalCode    string `json:"nok_postalcode"`
	NOKState         string `json:"nok_state"`
	NOKSurname       string `json:"nok_surname"`
	NOKTown          string `json:"nok_town"`
	OSpokenLang      string `json:"ospokenlang"`
	PFirstName       string `json:"pfirstname"`
	Photo            string `json:"photo"`
	PMiddleName      string `json:"pmiddlename"`
	Profession       string `json:"profession"`
	PSurname         string `json:"psurname"`
	Religion         string `json:"religion"`
	ResidenceAddress string `json:"residence_address"`
	ResidenceLGA     string `json:"residence_lga"`
	ResidenceState   string `json:"residence_state"`
	ResidenceTown    string `json:"residence_town"`
	ResidenceStatus  string `json:"residencestatus"`
	SelfOriginLGA    string `json:"self_origin_lga"`
	SelfOriginPlace  string `json:"self_origin_place"`
	SelfOriginState  string `json:"self_origin_state"`
	Signature        string `json:"signature"`
	SpokenLanguage   string `json:"spoken_language"`
	Surname          string `json:"surname"`
	TelephoneNo      string `json:"telephoneno"`
	Title            string `json:"title"`
	TrackingID       string `json:"trackingId"`
	UserID           string `json:"userid"`
	VNIN             string `json:"vnin"`
}

type NINVerificationResult struct {
	Success         bool   `json:"success"`          // Overall verification status
	NameMatched     bool   `json:"name_matched"`     // Whether names match
	ResponseCode    string `json:"response_code"`    // API response code
	ResponseMessage string `json:"response_message"` // API response message
	Reason          string `json:"reason,omitempty"` // Reason for failure (if any)
}
