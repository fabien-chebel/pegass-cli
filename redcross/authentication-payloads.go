package redcross

import "time"

type PasswordAuth struct {
	Password string              `json:"password"`
	Username string              `json:"username"`
	Options  PasswordAuthOptions `json:"options"`
}

type PasswordAuthOptions struct {
	WarnBeforePasswordExpired bool `json:"warnBeforePasswordExpired"`
	MultiOptionalFactorEnroll bool `json:"multiOptionalFactorEnroll"`
}

type MFAAuthRequest struct {
	PassCode   string `json:"passCode"`
	StateToken string `json:"stateToken"`
}

type MFAAuthResponse struct {
	Status       string `json:"status"`
	SessionToken string `json:"sessionToken"`
}

type PasswordAuthResponse struct {
	StateToken string    `json:"stateToken"`
	ExpiresAt  time.Time `json:"expiresAt"`
	Status     string    `json:"status"`
	Embedded   Embedded  `json:"_embedded"`
}

type Factors struct {
	ID         string `json:"id"`
	FactorType string `json:"factorType"`
	Provider   string `json:"provider"`
	VendorName string `json:"vendorName"`
}

type Embedded struct {
	Factors []Factors `json:"factors"`
}
