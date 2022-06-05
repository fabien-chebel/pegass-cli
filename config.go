package main

type Config struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	TotpSecretKey string `json:"totp_secret_key"`
}

type AuthTicket struct {
	CookieName  string `json:"name"`
	CookieValue string `json:"value"`
}
