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

	startDate := "2021-01-01"
	endDate := "2021-12-21"

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

func (pegassClient PegassClient) GetUserDetails(nivol string) (redcross.Utilisateur, error) {
	var user = redcross.Utilisateur{}
	var httpClient = http.Client{
		Jar: pegassClient.cookieJar,
	}

	requestURI := fmt.Sprintf("https://pegass.croix-rouge.fr/crf/rest/utilisateur/%s", nivol)

	getRequest, err := httpClient.Get(requestURI)
	if err != nil {
		return user, fmt.Errorf("failed to fetch user details: %w", err)
	}
	defer getRequest.Body.Close()

	err = json.NewDecoder(getRequest.Body).Decode(&user)
	if err != nil {
		return user, fmt.Errorf("failed to unmarshal response from Pegass: %w", err)
	}

	return user, nil
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
		if rechercheBenevoles.Total == 0 || rechercheBenevoles.Page == rechercheBenevoles.Total-1 {
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

func (pegassClient PegassClient) GetActivityStats() (map[string]redcross.RegulationStats, error) {
	var httpClient = http.Client{
		Jar: pegassClient.cookieJar,
	}

	startDate := "2021-01-01"
	endDate := "2021-12-31"

	parsedUri, err := url.Parse("https://pegass.croix-rouge.fr/crf/rest/seance")
	if err != nil {
		return nil, fmt.Errorf("failed to parse pegass API url: %w", err)
	}

	query := parsedUri.Query()
	query.Add("debut", startDate)
	query.Add("fin", endDate)
	query.Add("pageInfo", "true")
	query.Add("size", "100")
	query.Add("statut", "COMPLETE")
	query.Add("structure", "97")       // DT92
	query.Add("typeActivite", "10114") // Regulation

	currentPage := 0
	query.Add("page", strconv.Itoa(currentPage))

	parsedUri.RawQuery = query.Encode()

	var seanceIds []string

	for allResultsAreIn := false; !allResultsAreIn; {

		getRequest, err := httpClient.Get(parsedUri.String())
		if err != nil {
			return nil, fmt.Errorf("failed to create get request to pegass 'seance' endpoint: %w", err)
		}
		defer getRequest.Body.Close()

		var seanceList = redcross.SeanceList{}
		err = json.NewDecoder(getRequest.Body).Decode(&seanceList)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal search results: %s", err)
		}

		log.Printf("Parsing results for page %d / %d", currentPage+1, seanceList.TotalPages)

		for _, seance := range seanceList.Content {
			seanceIds = append(seanceIds, seance.ID)
		}

		if seanceList.Last == true {
			allResultsAreIn = true
		} else {
			currentPage++
			query.Set("page", strconv.Itoa(currentPage))
			parsedUri.RawQuery = query.Encode()
		}

	}

	var statsMap = make(map[string]redcross.RegulationStats)

	for _, id := range seanceIds {
		log.Printf("Computing stats for seance '%s'", id)

		inscriptionRequestURI := fmt.Sprintf("https://pegass.croix-rouge.fr/crf/rest/seance/%s/inscription", id)
		inscriptionRequest, err := httpClient.Get(inscriptionRequestURI)
		if err != nil {
			return nil, fmt.Errorf("failed to create request to pgeass 'seance' endpoint: %w", err)
		}
		defer inscriptionRequest.Body.Close()

		inscriptions := redcross.InscriptionList{}
		err = json.NewDecoder(inscriptionRequest.Body).Decode(&inscriptions)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal response from Pegass: %w", err)
		}

		for _, inscription := range inscriptions {
			entry, ok := statsMap[inscription.Utilisateur.ID]
			if !ok {
				entry = redcross.RegulationStats{
					OPR:   0,
					Eval:  0,
					Regul: 0,
				}
			}

			switch inscription.Role {
			case "47": // FORM OPR
				entry.OPR++
			case "18": // Régulateur
				entry.Regul++
			case "1": // Participant
				entry.OPR++
			case "80": // Aide-Régulateur
				entry.Regul++
			case "63": // Evaluateur régulateur
				entry.Eval++
			case "PARTICIPANT":
				entry.OPR++
			default:
				log.Printf("Unsupported role: %s ; seance id: %s", inscription.Role, inscription.Seance.ID)
			}

			statsMap[inscription.Utilisateur.ID] = entry
		}

	}

	return statsMap, nil
}

func (pegassClient PegassClient) GetUsersForRole(role redcross.Role) ([]redcross.Utilisateur, error) {
	var httpClient = http.Client{
		Jar: pegassClient.cookieJar,
	}

	parse, err := url.Parse("https://pegass.croix-rouge.fr/crf/rest/utilisateur")
	if err != nil {
		return nil, fmt.Errorf("failed to parse url to pegass: %w", err)
	}

	query := parse.Query()
	query.Add("pageInfo", "true")
	query.Add("size", "11")
	switch role.Type {
	case "COMP":
		query.Add("role", role.ID)
	case "NOMI":
		query.Add("nomination", role.ID)
	case "FORM":
		query.Add("formation", role.ID)
	default:
		log.Printf("Unsupported role type '%s'", role.Type)
		return nil, fmt.Errorf("unsupported role type '%s'", role.Type)
	}
	query.Add("searchType", "benevoles")
	query.Add("withMoyensCom", "true")
	query.Add("zoneGeoId", "92")
	query.Add("zoneGeoType", "departement")
	currentPage := 0
	currentPageAsString := strconv.Itoa(currentPage)
	query.Add("page", currentPageAsString)

	parse.RawQuery = query.Encode()

	var users []redcross.Utilisateur

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

		users = append(users, rechercheBenevoles.List...)

		// Should we exit the loop?
		if rechercheBenevoles.Total == 0 || rechercheBenevoles.Page == rechercheBenevoles.Total-1 {
			allResultsAreIn = true
		} else {
			currentPage++
			currentPageAsString = strconv.Itoa(currentPage)
			query.Set("page", currentPageAsString)
			parse.RawQuery = query.Encode()
		}

		log.Printf("Done parsing results for page %d", currentPage-1)
	}

	return users, nil
}

func (pegassClient PegassClient) FindRoleByName(roleName string) (redcross.Role, error) {
	var roles []redcross.Role

	var httpClient = http.Client{
		Jar: pegassClient.cookieJar,
	}

	getRequest, err := httpClient.Get("https://pegass.croix-rouge.fr/crf/rest/roles")
	if err != nil {
		return redcross.Role{}, fmt.Errorf("failed to create request to Pegass 'competences' endpoint: %w", err)
	}
	defer getRequest.Body.Close()

	err = json.NewDecoder(getRequest.Body).Decode(&roles)
	if err != nil {
		return redcross.Role{}, fmt.Errorf("failed to unmarshal search results: %w", err)
	}

	for _, role := range roles {
		if role.Libelle == roleName {
			return role, nil
		}
	}

	return redcross.Role{}, fmt.Errorf("failed to find any matching role")
}
