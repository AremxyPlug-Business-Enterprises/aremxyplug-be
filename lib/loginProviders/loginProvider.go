package loginProviders

// UserInfo is the user information retrieved from a Login Provider
// @TODO Check if we need more field than these ones
type UserInfo struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// JWTLoginProvider is a login provider that uses JWT as a means to share information. Google is an user of this approach
//
// jwtToken is the received string from the login provider
type JWTLoginProvider interface {
	UserInfo(jwtToken string) (*UserInfo, error)
}

// OAuth2LoginProvider is a login provider that uses OAuth2 as a means to communicate. Facebook and LinkedIn use this approach
type OAuth2LoginProvider interface {
	JWTLoginProvider
}
