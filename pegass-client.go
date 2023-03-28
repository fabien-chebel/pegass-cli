package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	redcross "github.com/fabien-chebel/pegass-cli/redcross"
	"github.com/pquerna/otp/totp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ActivityKind int

const (
	SAMU ActivityKind = iota
	BSPP
)

const (
	ACTIVITY_RESEAU_15_ID  = 10115
	ACTIVITY_RESEAU_18_ID  = 10116
	ACTIVITY_REGULATION_ID = 10114
)

type PegassClient struct {
	cookieJar     *cookiejar.Jar
	httpClient    *http.Client
	structures    map[int]string
	Username      string
	Password      string
	TotpSecretKey string
}

func (p *PegassClient) init() error {
	if p.cookieJar == nil {
		jar, err := cookiejar.New(nil)
		p.cookieJar = jar
		if err != nil {
			return fmt.Errorf("failed to create cookie jar: %w", err)
		}
	}
	p.httpClient = &http.Client{
		Jar: p.cookieJar,
	}
	return nil
}

func (p *PegassClient) kickOffAuthentication(username string, password string) (redcross.PasswordAuthResponse, error) {
	var passwordAuthResponse = redcross.PasswordAuthResponse{}

	passwordAuthPayload := redcross.PasswordAuth{
		Password: password,
		Username: username,
		Options: redcross.PasswordAuthOptions{
			WarnBeforePasswordExpired: true,
			MultiOptionalFactorEnroll: true,
		},
	}
	payloadBuffer := new(bytes.Buffer)
	err := json.NewEncoder(payloadBuffer).Encode(passwordAuthPayload)
	if err != nil {
		return passwordAuthResponse, fmt.Errorf("failed to encode password authentication payload: %w", err)
	}
	request, err := p.httpClient.Post("https://connect.croix-rouge.fr/api/v1/authn", "application/json", payloadBuffer)
	if err != nil {
		return passwordAuthResponse, fmt.Errorf("failed to send authentication request to Okta: %w", err)
	}
	defer request.Body.Close()
	log.WithFields(log.Fields{
		"statusCode": request.StatusCode,
	}).Debug("call to okta /api/v1/authn returned")

	err = json.NewDecoder(request.Body).Decode(&passwordAuthResponse)
	if err != nil {
		return passwordAuthResponse, fmt.Errorf("failed to decode password authentication response as json: %w", err)
	}

	return passwordAuthResponse, nil
}

func (p *PegassClient) obtainOktaSessionToken(factorId string, mfaCode string, stateToken string) (string, error) {
	payloadBuffer := new(bytes.Buffer)
	mfaRequest := redcross.MFAAuthRequest{
		PassCode:   mfaCode,
		StateToken: stateToken,
	}
	err := json.NewEncoder(payloadBuffer).Encode(mfaRequest)
	if err != nil {
		return "", fmt.Errorf("failed to encode MFA authentication request: %w", err)
	}

	request, err := p.httpClient.Post(
		fmt.Sprintf("https://connect.croix-rouge.fr/api/v1/authn/factors/%s/verify?rememberDevice=false", factorId),
		"application/json",
		payloadBuffer,
	)
	if err != nil {
		return "", fmt.Errorf("failed to send MFA challenge response: %w", err)
	}
	defer request.Body.Close()

	var mfaAuthResponse = redcross.MFAAuthResponse{}
	err = json.NewDecoder(request.Body).Decode(&mfaAuthResponse)
	if err != nil {
		return "", fmt.Errorf("failed to parse MFA validation response: %w", err)
	}
	log.WithFields(log.Fields{
		"sessionToken": mfaAuthResponse.SessionToken,
	}).Debug("successfully obtained session token")

	return mfaAuthResponse.SessionToken, nil
}

func (p *PegassClient) Authenticate() error {
	err := p.init()
	if err != nil {
		return err
	}

	passwordAuthResponse, err := p.kickOffAuthentication(p.Username, p.Password)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"status": passwordAuthResponse.Status,
	}).Debug("password authentication request successfully performed")
	if passwordAuthResponse.Status != "MFA_REQUIRED" {
		return fmt.Errorf("expected Okta to ask for MFA challenge but instead got status '%s'", passwordAuthResponse.Status)
	}

	factorId, err := findTotpFactorId(passwordAuthResponse.Embedded.Factors)
	if err != nil {
		return fmt.Errorf("unable to find any TOTP generator registered to this account: %w", err)
	}

	code, err := totp.GenerateCode(p.TotpSecretKey, time.Now())
	if err != nil {
		return fmt.Errorf("failed to generate TOTP code: %w", err)
	}
	log.WithFields(log.Fields{
		"code": code,
	}).Debug("generated 2FA totp code")

	sessionToken, err := p.obtainOktaSessionToken(factorId, code, passwordAuthResponse.StateToken)

	request, err := p.httpClient.Get(fmt.Sprintf("https://connect.croix-rouge.fr/home/croix-rouge_pegass_1/0oa2s6fw19Pp8eQzd417/aln2s6knvxzI5pG6x417?sessionToken=%s", sessionToken))
	if err != nil {
		return fmt.Errorf("failed to authenticate to Pegass: %w", err)
	}
	defer request.Body.Close()
	var samlResponseToken string
	tokenizer := html.NewTokenizer(request.Body)
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

	authentRequest, err := p.httpClient.PostForm("https://pegass.croix-rouge.fr/Shibboleth.sso/SAML2/POST", url.Values{
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
	pegassCookies := p.cookieJar.Cookies(pegassUrl)
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

func findTotpFactorId(factors []redcross.Factors) (string, error) {
	for _, factor := range factors {
		if factor.FactorType == "token:software:totp" {
			log.WithFields(log.Fields{
				"factorType": factor.FactorType,
				"factorId":   factor.ID,
			}).Debug("found totp compatible mutlti-factor generator")
			return factor.ID, nil
		}
	}
	return "", errors.New("no totp generator associated with your account")
}

func (p *PegassClient) ReAuthenticate() error {
	err := p.init()
	if err != nil {
		return err
	}

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

	p.cookieJar = jar
	return nil
}

func (p *PegassClient) AuthenticateIfNecessary() error {
	if !p.shouldReAuthenticate() {
		log.Info("shouldReAuthenticated returned false")
		return nil
	}
	log.Info("previous authentication ticket expired. application will re-authenticate to pegass")
	p.cookieJar = nil

	return p.Authenticate()
}

func (p *PegassClient) shouldReAuthenticate() bool {
	noRedirectHttpClient := &http.Client{
		Jar: p.cookieJar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	response, err := noRedirectHttpClient.Get("https://pegass.croix-rouge.fr/crf/rest/gestiondesdroits")
	if err != nil {
		log.Warnf("reauthenticate check request failed: '%s'", err.Error())
		return true
	}
	defer response.Body.Close()

	log.Infof("Call to /gestiondesdroits endpoint returned response with code '%d' and headers '%s'", response.StatusCode, response.Header)
	return response.StatusCode != http.StatusOK
}

func (p *PegassClient) GetCurrentUser() (redcross.GestionDesDroits, error) {
	var user = redcross.GestionDesDroits{}
	err := p.init()
	if err != nil {
		return user, err
	}

	getRequest, err := p.httpClient.Get("https://pegass.croix-rouge.fr/crf/rest/gestiondesdroits")
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

func (p *PegassClient) GetStatsForUser(nivol string) (redcross.StatsBenevole, error) {
	var stats = redcross.StatsBenevole{}
	err := p.init()
	if err != nil {
		return stats, err
	}

	startDate := "2021-01-01"
	endDate := "2021-12-21"

	requestURI := fmt.Sprintf("https://pegass.croix-rouge.fr/crf/rest/statistiques/benevole/%s/%s/%s/quantite", nivol, startDate, endDate)
	getRequest, err := p.httpClient.Get(requestURI)
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

func (p *PegassClient) GetUserDetails(nivol string) (redcross.Utilisateur, error) {
	var user = redcross.Utilisateur{}
	err := p.init()
	if err != nil {
		return user, err
	}

	requestURI := fmt.Sprintf("https://pegass.croix-rouge.fr/crf/rest/utilisateur/%s", nivol)

	getRequest, err := p.httpClient.Get(requestURI)
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

func (p *PegassClient) GetDispatchers() ([]redcross.Utilisateur, error) {
	const DISPATCHER_ROLE_ID = "18"
	err := p.init()
	if err != nil {
		return nil, err
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
		getRequest, err := p.httpClient.Get(parse.String())
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

func (p *PegassClient) GetActivityStats() (map[string]redcross.RegulationStats, error) {
	err := p.init()
	if err != nil {
		return nil, err
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

		getRequest, err := p.httpClient.Get(parsedUri.String())
		if err != nil {
			return nil, fmt.Errorf("failed to create get request to pegass 'seance' endpoint: %w", err)
		}
		defer getRequest.Body.Close()

		var seanceList = redcross.SeanceList{}
		err = json.NewDecoder(getRequest.Body).Decode(&seanceList)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal search results: %s", err)
		}

		log.Infof("Parsing results for page %d / %d", currentPage+1, seanceList.TotalPages)

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
		log.Infof("Computing stats for seance '%s'", id)

		inscriptionRequestURI := fmt.Sprintf("https://pegass.croix-rouge.fr/crf/rest/seance/%s/inscription", id)
		inscriptionRequest, err := p.httpClient.Get(inscriptionRequestURI)
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

func (p *PegassClient) GetMainMoyenComForUser(nivol string) (string, error) {
	response, err := p.httpClient.Get(fmt.Sprintf("https://pegass.croix-rouge.fr/crf/rest/moyencomutilisateur?utilisateur=%s", nivol))
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	var moyenComs []redcross.Coordonnees
	err = json.NewDecoder(response.Body).Decode(&moyenComs)
	if err != nil {
		return "", err
	}

	for _, com := range moyenComs {
		if com.MoyenComID == "POR" {
			return com.Libelle, nil
		}
	}

	return "", nil
}

func (p *PegassClient) GetUsersForRole(role redcross.Role) ([]redcross.Utilisateur, error) {
	err := p.init()
	if err != nil {
		return nil, err
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
		getRequest, err := p.httpClient.Get(parse.String())
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

func (p *PegassClient) GetAllStructuresForDepartment(department string) ([]int, error) {
	structures, err := p.GetStructuresForDepartment(department)
	if err != nil {
		return nil, err
	}

	var ids []int
	for key, _ := range structures {
		ids = append(ids, key)
	}
	return ids, nil
}

func (p *PegassClient) GetStructuresForDepartment(department string) (map[int]string, error) {
	err := p.init()
	if err != nil {
		return nil, err
	}

	request, err := p.httpClient.Get(fmt.Sprintf("https://pegass.croix-rouge.fr/crf/rest/zonegeo/departement/%s", department))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the list of department structures: %w", err)
	}
	defer request.Body.Close()

	var structureList = redcross.StructureList{}
	err = json.NewDecoder(request.Body).Decode(&structureList)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize pegass request: %w", err)
	}

	var dict = make(map[int]string)
	for _, structure := range structureList.StructuresFilles {
		filteredName := strings.ReplaceAll(structure.Libelle, "UNITE LOCALE DE ", "")
		filteredName = strings.ReplaceAll(filteredName, "UNITE LOCALE D'", "")
		dict[structure.ID] = filteredName
	}

	return dict, nil
}

func (p *PegassClient) GetUsersForTrainingRole(role redcross.Role) ([]redcross.Utilisateur, error) {
	err := p.init()
	if err != nil {
		return nil, err
	}

	structures, err := p.GetAllStructuresForDepartment("92")
	if err != nil {
		return nil, err
	}

	parse, err := url.Parse("https://pegass.croix-rouge.fr/crf/rest/utilisateur/advancedSearch")
	if err != nil {
		return nil, fmt.Errorf("failed to parse url to pegass: %w", err)
	}

	query := parse.Query()
	query.Add("size", "11")
	currentPage := 0
	currentPageAsString := strconv.Itoa(currentPage)
	query.Add("page", currentPageAsString)

	parse.RawQuery = query.Encode()

	jsonBody := redcross.AdvancedSearch{
		StructureList:   structures,
		FormationInList: []string{role.ID},
		SearchType:      "benevoles",
		WithMoyensCom:   true,
	}
	payloadBuffer := new(bytes.Buffer)
	err = json.NewEncoder(payloadBuffer).Encode(jsonBody)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize search parameters: %w", err)
	}

	var users []redcross.Utilisateur

	for allResultsAreIn := false; !allResultsAreIn; {
		request, err := p.httpClient.Post(parse.String(), "application/json", payloadBuffer)
		if err != nil {
			return nil, fmt.Errorf("failed to execute advanced pegass search: %w", err)
		}
		defer request.Body.Close()

		var rechercheBenevoles = redcross.RechercheBenevoles{}
		err = json.NewDecoder(request.Body).Decode(&rechercheBenevoles)
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

func (p *PegassClient) FindRoleByName(roleName string) (redcross.Role, error) {
	var roles []redcross.Role

	err := p.init()
	if err != nil {
		return redcross.Role{}, err
	}

	getRequest, err := p.httpClient.Get("https://pegass.croix-rouge.fr/crf/rest/roles")
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

func (p *PegassClient) lintActivity(activity redcross.Activity) (string, error) {
	var inscriptionUrl = fmt.Sprintf("https://pegass.croix-rouge.fr/crf/rest/seance/%s/inscription", activity.SeanceList[0].ID)
	response, err := p.httpClient.Get(inscriptionUrl)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	inscriptions := redcross.InscriptionList{}
	err = json.NewDecoder(response.Body).Decode(&inscriptions)
	var chiefContactDetails string
	var hasFormerFirstResponder bool

	var dispatcherAssociation string
	var minorCount, chiefCount, driverCount, pse2Count, pse1Count, traineeCount, dispatcherCount, dispatcherTrainerCount, radioOperatorCount, unknownCount int
	for _, inscription := range inscriptions {
		userDetails, err := p.GetUserDetails(inscription.Utilisateur.ID)
		if err != nil {
			return "", err
		}
		phoneNumber, err := p.GetMainMoyenComForUser(inscription.Utilisateur.ID)
		if err != nil {
			log.Warnf("failed to fetch phone number of user '%s'", inscription.Utilisateur.ID)
		}
		if userDetails.Mineur {
			minorCount++
		}

		if inscription.Type == "NOMI" && (inscription.Role == "254" || inscription.Role == "255") {
			// CI RESEAU || CI BSPP
			chiefCount++
			if phoneNumber == "" {
				phoneNumber = "(Inconnu)"
			}
			chiefContactDetails = fmt.Sprintf("📞 %s %s %s", userDetails.Prenom, userDetails.Nom, phoneNumber)
		} else if inscription.Type == "COMP" && inscription.Role == "10" {
			// CH
			driverCount++
		} else if inscription.Type == "FORM" && inscription.Role == "167" {
			// PSE2
			pse2Count++
		} else if inscription.Type == "FORM" && inscription.Role == "166" {
			pse1Count++
		} else if inscription.Role == "PARTICIPANT" {
			traineeCount++
			isFormerFirstResponder, err := p.IsFormerFirstResponder(inscription.Utilisateur.ID)
			if err != nil {
				log.Warnf("failed to check whether user '%s' used to be a first responder: %v", inscription.Utilisateur.ID, err)
			}
			hasFormerFirstResponder = hasFormerFirstResponder || isFormerFirstResponder
		} else if inscription.Type == "COMP" && (inscription.Role == "18" || inscription.Role == "80" || inscription.Role == "63") {
			dispatcherCount++
			assoc, ok := EXTERNAL_ASSOCIATIONS[inscription.Utilisateur.ID]
			if ok {
				dispatcherAssociation = assoc
			} else {
				dispatcherAssociation = "CRF"
			}

			if inscription.Role == "63" {
				dispatcherTrainerCount++
			}
		} else if inscription.Type == "FORM" && inscription.Role == "47" {
			radioOperatorCount++
		} else {
			log.WithFields(log.Fields{
				"libelle":    activity.Libelle,
				"activityId": activity.ID,
				"role":       inscription.Role,
				"roleType":   inscription.Type,
				"nivol":      inscription.Utilisateur.ID,
			}).Warnf("came accross unknown role for activity '%s' and start date '%s'", activity.Libelle, time.Time(activity.SeanceList[0].Debut))
			unknownCount++
		}
	}

	buf := new(bytes.Buffer)
	if len(inscriptions) == 0 {
		return "[0 PAX]", nil
	}

	if activity.Libelle == "REGULATION" {
		if dispatcherAssociation != "" {
			buf.WriteString(fmt.Sprintf("[%d PAX][%s]\n\t\t%d ARS, %d OPR, %d Stagiaire", len(inscriptions), dispatcherAssociation, dispatcherCount, radioOperatorCount, traineeCount))
		} else {
			buf.WriteString(fmt.Sprintf("[%d PAX]\n\t\t%d ARS, %d OPR, %d Stagiaire", len(inscriptions), dispatcherCount, radioOperatorCount, traineeCount))
		}
		if dispatcherTrainerCount > 0 {
			buf.WriteString("\n\t\tℹ️Evaluation régulateur")
		}
	} else {
		buf.WriteString(fmt.Sprintf("[%d PAX]", len(inscriptions)))
	}
	if minorCount > 0 {
		buf.WriteString(fmt.Sprintf("\n\t\t⚠️ %d 🔞", minorCount))
	}
	if chiefContactDetails != "" {
		buf.WriteString("\n\t\t" + chiefContactDetails)
	}

	if activity.Libelle != "REGULATION" {
		if pse1Count > 1 {
			buf.WriteString(fmt.Sprintf("\n\t\t⚠️ %d PSE1 (max 1)", pse1Count))
		}
		if pse2Count == 0 {
			buf.WriteString(fmt.Sprintf("\n\t\t⚠️ Aucun PSE2"))
		}
		if hasFormerFirstResponder {
			buf.WriteString(fmt.Sprintf("\n\t\t⚠️ Observateur PSE non-recyclé"))
		}
	}

	return buf.String(), nil
}

func (p *PegassClient) FindActivitiesOnDay(day string, kind ActivityKind, shouldCensorData bool) (string, error) {
	err := p.init()
	if err != nil {
		return "", err
	}

	parse, err := url.Parse("https://pegass.croix-rouge.fr/crf/rest/seance")
	if err != nil {
		return "", fmt.Errorf("failed to parse url to pegass: %w", err)
	}

	query := parse.Query()
	query.Add("action", "65")
	query.Add("debut", day)
	query.Add("fin", day)
	query.Add("page", "0")
	query.Add("pageInfo", "true")
	query.Add("size", "100")
	query.Add("zoneGeoId", "92")
	query.Add("zoneGeoType", "departement")

	parse.RawQuery = query.Encode()

	request, err := p.httpClient.Get(parse.String())
	if err != nil {
		return "", fmt.Errorf("failed to search for activities: %w", err)
	}
	defer request.Body.Close()

	var seanceList = redcross.SeanceList{}
	err = json.NewDecoder(request.Body).Decode(&seanceList)
	if err != nil {
		return "", fmt.Errorf("failed to deserialize seance list: %w", err)
	}

	var activities []redcross.Activity
	for _, seance := range seanceList.Content {
		activity, err := p.fetchActivityById(seance.Activite.ID)
		if err != nil {
			log.Warnf("unable to map seance '%s' to activity: %s", seance.ID, err)
		}
		activities = append(activities, activity)
	}

	summary, err := p.summarize(activities, kind, shouldCensorData)
	if err != nil {
		return "", fmt.Errorf("failed to summarize activities: %w", err)
	}

	return summary, nil

}

func (p *PegassClient) fetchActivityById(activityId string) (redcross.Activity, error) {
	activity := redcross.Activity{}

	url := fmt.Sprintf("https://pegass.croix-rouge.fr/crf/rest/activite/%s", activityId)
	request, err := p.httpClient.Get(url)
	if err != nil {
		return activity, fmt.Errorf("failed to search for activities: %w", err)
	}
	defer request.Body.Close()

	err = json.NewDecoder(request.Body).Decode(&activity)
	if err != nil {
		return activity, fmt.Errorf("failed to deserialize pegass activity: %w", err)
	}

	return activity, nil
}

func (p *PegassClient) summarize(activities []redcross.Activity, kind ActivityKind, shouldCensorData bool) (string, error) {
	sort.Sort(redcross.ByActivity(activities))
	department, err := p.GetStructuresForDepartment("92")
	if err != nil {
		return "", err
	}
	p.structures = department

	var buffer bytes.Buffer
	var previousActivity string
	for _, act := range activities {
		if act.StructureMenantActivite.ID == 0 || act.TypeActivite.Action.ID != 65 {
			// Skip unaffected activities and !"Réseau de secours"
			continue
		}

		if kind == SAMU {
			if act.TypeActivite.ID != ACTIVITY_RESEAU_15_ID && act.TypeActivite.ID != ACTIVITY_REGULATION_ID {
				// Only keep REGULATION and SAMU activities
				continue
			}
		} else if kind == BSPP {
			if act.TypeActivite.ID != ACTIVITY_RESEAU_18_ID {
				continue
			}
		}

		var isCRFActivity = true
		if _, ok := EXTERNAL_ASSOCIATIONS[act.Responsable.ID]; ok {
			isCRFActivity = false
		}

		var comment string
		if isCRFActivity && !shouldCensorData {
			comment, err = p.lintActivity(act)
			if err != nil {
				return "", err
			}
		}

		if previousActivity != act.Libelle {
			// Create a section
			var structInfo string
			if isCRFActivity && act.StructureMenantActivite.ID != 0 && act.Libelle != "REGULATION" {
				structInfo = p.structures[act.StructureMenantActivite.ID]
			} else if !isCRFActivity {
				structInfo = EXTERNAL_ASSOCIATIONS[act.Responsable.ID]
			}
			buffer.WriteString(fmt.Sprintf("\n%s", act.Libelle))
			if structInfo != "" {
				buffer.WriteString(fmt.Sprintf(" [%s]", structInfo))
			}
			buffer.WriteString("\n")
		}
		seance := act.SeanceList[0]
		state := fmt.Sprintf("\t%s — %s - %s %s\n", mapStatusToEmoji(act.Statut), seance.Debut.PrintTimePart(), seance.Fin.PrintTimePart(), comment)
		buffer.WriteString(state)

		previousActivity = act.Libelle
	}

	return buffer.String(), nil
}

func (p *PegassClient) GetTrainingsForUser(nivol string) ([]redcross.UserTraining, error) {
	parse, err := url.Parse("https://pegass.croix-rouge.fr/crf/rest/formationutilisateur")
	if err != nil {
		return nil, fmt.Errorf("failed to parse url to pegass: %v", err)
	}

	query := parse.Query()
	query.Add("utilisateur", nivol)
	parse.RawQuery = query.Encode()

	request, err := p.httpClient.Get(parse.String())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user trainings: %v", err)
	}
	defer request.Body.Close()

	var trainings []redcross.UserTraining
	err = json.NewDecoder(request.Body).Decode(&trainings)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize training list: %v", err)
	}

	return trainings, nil
}

func (p *PegassClient) IsFormerFirstResponder(nivol string) (bool, error) {
	trainings, err := p.GetTrainingsForUser(nivol)
	if err != nil {
		return false, err
	}

	for _, t := range trainings {
		if (t.Formation.Code == "PSE2" || t.Formation.Code == "PSE1") && !t.Formation.Recyclage {
			return true, nil
		}
	}

	return false, nil
}

var EXTERNAL_ASSOCIATIONS = map[string]string{
	"01100009671G": "PCPS",
	"01100009672H": "Malte",
	"01100039741E": "FFSS",
}

func mapStatusToEmoji(status string) string {
	switch status {
	case "Complète":
		return "✅ "
	case "Incomplète":
		return "❌ "
	case "Annulée":
		return "🟡"
	default:
		return "?"
	}
}
