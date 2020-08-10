package main

import (
	"encoding/json"
	"errors"
	"fmt"
	redcross "github.com/fabien-chebel/pegass-cli/redcross"
	"golang.org/x/net/html"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
)

type PegassClient struct {
	cookieJar *cookiejar.Jar
}

func (pegassClient *PegassClient) Authenticate(username string, password string) error {

	if pegassClient.cookieJar == nil {
		jar, err := cookiejar.New(nil)
		pegassClient.cookieJar = jar
		if err != nil {
			return fmt.Errorf("failed to create cookie jar: %w", err)
		}
	}

	client := http.Client{
		Jar: pegassClient.cookieJar,
	}

	get, err := client.Get("https://pegass.croix-rouge.fr/")
	if err != nil {
		return fmt.Errorf("failed to process request: %w", err)
	}
	defer get.Body.Close()
	_, err = ioutil.ReadAll(get.Body)
	if err != nil {
		return fmt.Errorf("failed to access Pegass: %w", err)
	}

	formRequest, err := client.PostForm("https://id.authentification.croix-rouge.fr/my.policy", url.Values{
		"username": {username},
		"password": {password},
		"vhost":    {"standard"},
	})

	if err != nil {
		return fmt.Errorf("failed to authenticate to Pegass: %w", err)
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
		return errors.New("failed to parse SAML Response token")
	}

	authentRequest, err := client.PostForm("https://pegass.croix-rouge.fr/Shibboleth.sso/SAML2/POST", url.Values{
		"SAMLResponse": {samlResponseToken},
	})
	if err != nil {
		return fmt.Errorf("failed to authenticate on Pegass: %w", err)
	}
	defer authentRequest.Body.Close()

	pegassUrl, err := url.Parse("https://pegass.croix-rouge.fr")
	if err != nil {
		return fmt.Errorf("failed to parse pegass URL: %w", err)
	}
	pegassCookies := pegassClient.cookieJar.Cookies(pegassUrl)
	if len(pegassCookies) != 1 {
		return errors.New("error: expected to find a single Cookie for Pegass domain")
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
		return fmt.Errorf("failed to serialize authentication data: %w", err)
	}
	err = ioutil.WriteFile("auth-ticket.json", fileContent, 0700)
	if err != nil {
		return fmt.Errorf("failed to save authentication data to file: %w", err)
	}

	log.Println("Authentication succeeded.")
	return nil
}

func (pegassClient *PegassClient) ReAuthenticate() error {
	file, err := os.Open("auth-ticket.json")
	if err != nil {
		return fmt.Errorf("failed to read authentication ticket file: %w", err)
	}

	var authTicket = AuthTicket{}
	err = json.NewDecoder(file).Decode(&authTicket)
	if err != nil {
		return fmt.Errorf("failed to deserialize authentication data: %w", err)
	}

	jar, _ := cookiejar.New(nil)
	pegassUrl, err := url.Parse("https://pegass.croix-rouge.fr")
	if err != nil {
		return fmt.Errorf("failed to parse pegass url: %w", err)
	}

	cookies := []*http.Cookie{{
		Name:  authTicket.CookieName,
		Value: authTicket.CookieValue,
	}}
	jar.SetCookies(pegassUrl, cookies)

	pegassClient.cookieJar = jar
	return nil
}

func (pegassClient PegassClient) GetCurrentUser() (redcross.GestionDesDroits, error) {
	var user = redcross.GestionDesDroits{}
	var httpClient = http.Client{
		Jar: pegassClient.cookieJar,
	}

	getRequest, err := httpClient.Get("https://pegass.croix-rouge.fr/crf/rest/gestiondesdroits")
	if err != nil {
		return user, fmt.Errorf("failed to create request to Pegass 'gestiondesdroits' endpoint: %w", err)
	}
	defer getRequest.Body.Close()

	err = json.NewDecoder(getRequest.Body).Decode(&user)
	if err != nil {
		return user, fmt.Errorf("failed to unmarshal response from Pegass: %w", err)
	}

	return user, nil
}

func (pegassClient PegassClient) GetStatsForUser(nivol string) (redcross.StatsBenevole, error) {
	var stats = redcross.StatsBenevole{}
	var httpClient = http.Client{
		Jar: pegassClient.cookieJar,
	}

	startDate := "2020-01-01"
	endDate := "2020-08-31"

	requestURI := fmt.Sprintf("https://pegass.croix-rouge.fr/crf/rest/statistiques/benevole/%s/%s/%s/quantite", nivol, startDate, endDate)
	getRequest, err := httpClient.Get(requestURI)
	if err != nil {
		return stats, fmt.Errorf("failed to create request to pegass 'statistiques benevole' endpoint: %w", err)
	}
	defer getRequest.Body.Close()

	stats = redcross.StatsBenevole{}
	err = json.NewDecoder(getRequest.Body).Decode(&stats)
	if err != nil {
		return stats, fmt.Errorf("failed to unmarshal response from Pegass: %w", err)
	}

	return stats, nil
}

func (pegassClient PegassClient) GetDispatchers() ([]redcross.Utilisateur, error) {
	const DISPATCHER_ROLE_ID = "18"

	var httpClient = http.Client{
		Jar: pegassClient.cookieJar,
	}

	parse, err := url.Parse("https://pegass.croix-rouge.fr/crf/rest/utilisateur")
	if err != nil {
		return nil, fmt.Errorf("failed to parse url to pegass: %w", err)
	}

	query := parse.Query()
	query.Add("pageInfo", "true")
	query.Add("perPage", "11")
	query.Add("role", DISPATCHER_ROLE_ID)
	query.Add("searchType", "benevoles")
	query.Add("withMoyensCom", "true")
	query.Add("zoneGeoId", "92")
	query.Add("zoneGeoType", "departement")
	currentPage := 0
	currentPageAsString := strconv.Itoa(currentPage)
	query.Add("page", currentPageAsString)

	parse.RawQuery = query.Encode()

	var dispatchers []redcross.Utilisateur

	for allResultsAreIn := false; !allResultsAreIn; {
		getRequest, err := httpClient.Get(parse.String())
		if err != nil {
			return nil, fmt.Errorf("failed to create request to pegass 'recherche utilisateur' endpoint: %w", err)
		}
		defer getRequest.Body.Close()

		var rechercheBenevoles = redcross.RechercheBenevoles{}
		err = json.NewDecoder(getRequest.Body).Decode(&rechercheBenevoles)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal search results: %w", err)
		}

		dispatchers = append(dispatchers, rechercheBenevoles.List...)

		// Should we exit the loop?
		if rechercheBenevoles.Pages == 0 || rechercheBenevoles.Page == rechercheBenevoles.Pages-1 {
			allResultsAreIn = true
		} else {
			currentPage++
			currentPageAsString = strconv.Itoa(currentPage)
			query.Set("page", currentPageAsString)
			parse.RawQuery = query.Encode()
		}

		log.Printf("Done parsing results for page %d", currentPage-1)
	}

	return dispatchers, nil

}
