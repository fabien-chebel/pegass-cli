package main

import (
	"encoding/json"
	"golang.org/x/net/html"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
)

type PegassClient struct {
	cookieJar *cookiejar.Jar
}

func (pegassClient *PegassClient) Authenticate(username string, password string) {
	if pegassClient.cookieJar == nil {
		jar, err := cookiejar.New(nil)
		pegassClient.cookieJar = jar
		if err != nil {
			log.Fatal("Failed to create cookie jar", err)
		}
	}

	client := http.Client{
		Jar: pegassClient.cookieJar,
	}

	get, err := client.Get("https://pegass.croix-rouge.fr/")
	if err != nil {
		log.Fatal("Failed to process request", err)
	}
	defer get.Body.Close()
	_, err = ioutil.ReadAll(get.Body)
	if err != nil {
		log.Fatal("Failed to access Pegass", err)
	}

	formRequest, err := client.PostForm("https://id.authentification.croix-rouge.fr/my.policy", url.Values{
		"username": {username},
		"password": {password},
		"vhost":    {"standard"},
	})

	if err != nil {
		log.Fatal("Failed to authenticate to Pegass", err)
	}
	defer formRequest.Body.Close()
	var samlResponseToken string
	tokenizer := html.NewTokenizer(formRequest.Body)
tokenLoop:
	for {
		tokenType := tokenizer.Next()
		switch {
		case tokenType == html.ErrorToken:
			break tokenLoop
		case tokenType == html.SelfClosingTagToken:
			token := tokenizer.Token()
			if token.Data == "input" {
				for _, a := range token.Attr {
					if a.Key == "name" && a.Val == "SAMLResponse" {
						for _, aa := range token.Attr {
							if aa.Key == "value" {
								samlResponseToken = aa.Val
							}
						}
					}
				}
			}
		}
	}

	if samlResponseToken == "" {
		log.Fatal("Failed to parse SAML Response token")
	}

	authentRequest, err := client.PostForm("https://pegass.croix-rouge.fr/Shibboleth.sso/SAML2/POST", url.Values{
		"SAMLResponse": {samlResponseToken},
	})
	if err != nil {
		log.Fatal("Failed to authenticate on Pegass", err)
	}
	defer authentRequest.Body.Close()

	pegassUrl, err := url.Parse("https://pegass.croix-rouge.fr")
	if err != nil {
		log.Fatal("Failed to parse pegass URL", err)
	}
	pegassCookies := pegassClient.cookieJar.Cookies(pegassUrl)
	if len(pegassCookies) != 1 {
		log.Fatal("Error: expected to find a single Cookie for Pegass domain")
	}
	pegassCookie := pegassCookies[0]

	cookieName := pegassCookie.Name
	cookieValue := pegassCookie.Value
	ticket := AuthTicket{
		CookieName:  cookieName,
		CookieValue: cookieValue,
	}

	fileContent, err := json.MarshalIndent(ticket, "", "")
	if err != nil {
		log.Fatal("Failed to serialize authentication data", err)
	}
	err = ioutil.WriteFile("auth-ticket.json", fileContent, 0700)
	if err != nil {
		log.Fatal("Failed to save authentication data to file", err)
	}

	log.Println("Authentication succeeded.")
}

func (pegassClient *PegassClient) ReAuthenticate() {
	file, err := os.Open("auth-ticket.json")
	if err != nil {
		log.Fatal("Failed to read authentication ticket file", err)
	}

	var authTicket = AuthTicket{}
	err = json.NewDecoder(file).Decode(&authTicket)
	if err != nil {
		log.Fatal("Failed to deserialize authentication data", err)
	}

	jar, _ := cookiejar.New(nil)
	pegassUrl, _ := url.Parse("https://pegass.croix-rouge.fr")

	cookies := []*http.Cookie{{
		Name:  authTicket.CookieName,
		Value: authTicket.CookieValue,
	}}
	jar.SetCookies(pegassUrl, cookies)

	pegassClient.cookieJar = jar
}

func (pegassClient PegassClient) GetCurrentUser() {
	var httpClient = http.Client{
		Jar: pegassClient.cookieJar,
	}

	getRequest, err := httpClient.Get("https://pegass.croix-rouge.fr/crf/rest/gestiondesdroits")
	if err != nil {
		log.Fatal("Failed to create request to Pegass 'gestiondesdroits' endpoint", err)
	}
	defer getRequest.Body.Close()

	var utilisateur = GestionDesDroits{}
	err = json.NewDecoder(getRequest.Body).Decode(&utilisateur)
	if err != nil {
		log.Fatal("Failed to unmarshal response from Pegass", err)
	}

	log.Printf("Bonjour %s %s (NIVOL: %s) !", utilisateur.Utilisateur.Prenom, utilisateur.Utilisateur.Nom, utilisateur.Utilisateur.Id)

}
