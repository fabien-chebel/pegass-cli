package main

type Config struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthTicket struct {
	CookieName  string `json:"name"`
	CookieValue string `json:"value"`
}
