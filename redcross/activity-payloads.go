package redcross

import (
	"strings"
	"time"
)

type SeanceList struct {
	Content          []Seance    `json:"content"`
	Last             bool        `json:"last"`
	TotalElements    int         `json:"totalElements"`
	TotalPages       int         `json:"totalPages"`
	Size             int         `json:"size"`
	Number           int         `json:"number"`
	Sort             interface{} `json:"sort"`
	First            bool        `json:"first"`
	NumberOfElements int         `json:"numberOfElements"`
}

type Seance struct {
	ID       string `json:"id"`
	Activite struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Libelle string `json:"libelle"`
	} `json:"activite"`
	GroupeAction struct {
		ID       int    `json:"id"`
		Libelle  string `json:"libelle"`
		Tri      string `json:"tri"`
		CanAdmin bool   `json:"canAdmin"`
	} `json:"groupeAction"`
	Debut          PegassTime `json:"debut"`
	Fin            PegassTime `json:"fin"`
	Adresse        string     `json:"adresse"`
	RevisionNumber int        `json:"revisionNumber"`
	RoleConfigList []struct {
		ID       string `json:"id"`
		Code     string `json:"code"`
		Role     string `json:"role"`
		Actif    bool   `json:"actif"`
		Effectif int    `json:"effectif"`
		Type     string `json:"type"`
	} `json:"roleConfigList"`
}

type InscriptionList []struct {
	ID       string `json:"id"`
	Activite struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		TypeActivite struct {
			ID           int    `json:"id"`
			Libelle      string `json:"libelle"`
			GroupeAction struct {
				ID       int    `json:"id"`
				Libelle  string `json:"libelle"`
				Tri      string `json:"tri"`
				CanAdmin bool   `json:"canAdmin"`
			} `json:"groupeAction"`
			Action struct {
				ID           int    `json:"id"`
				Libelle      string `json:"libelle"`
				GroupeAction struct {
					ID      int    `json:"id"`
					Libelle string `json:"libelle"`
				} `json:"groupeAction"`
			} `json:"action"`
			CanAdmin bool `json:"canAdmin"`
		} `json:"typeActivite"`
		Statut string `json:"statut"`
	} `json:"activite"`
	Seance struct {
		ID string `json:"id"`
	} `json:"seance"`
	Utilisateur Utilisateur `json:"utilisateur"`
	Debut       string      `json:"debut"`
	Fin         string      `json:"fin"`
	Statut      string      `json:"statut"`
	Role        string      `json:"role"`
	Type        string      `json:"type"`
	IsMultiple  bool        `json:"isMultiple"`
}

type Activity struct {
	ID                      string       `json:"id"`
	Libelle                 string       `json:"libelle"`
	StructureOrganisatrice  Structure    `json:"structureOrganisatrice"`
	StructureMenantActivite Structure    `json:"structureMenantActivite"`
	Statut                  string       `json:"statut"`
	SeanceList              []Seance     `json:"seanceList"`
	TypeActivite            TypeActivite `json:"typeActivite"`
	Responsable             Utilisateur  `json:"responsable"`
}

type TypeActivite struct {
	ID      int    `json:"id"`
	Libelle string `json:"libelle"`
	Action  Action `json:"action"`
}

type Action struct {
	ID      int    `json:"id"`
	Libelle string `json:"libelle"`
}

type ByActivity []Activity

func (a ByActivity) Len() int {
	return len(a)
}

func (a ByActivity) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByActivity) Less(i, j int) bool {
	// DAUPHIN < CASTOR < RUBIS < SAPHIR < BABETTE < REGULATION
	var first = a[i]
	var second = a[j]

	if ACTIVITY_MAP[first.Libelle] == ACTIVITY_MAP[second.Libelle] {
		if len(first.SeanceList) > 0 && len(second.SeanceList) > 0 {
			return time.Time(first.SeanceList[0].Debut).Before(time.Time(second.SeanceList[0].Debut))
		} else {
			return true
		}
	} else {
		return ACTIVITY_MAP[first.Libelle] < ACTIVITY_MAP[second.Libelle]
	}

}

var ACTIVITY_MAP = map[string]int{"01-DAUPHIN": 1,
	"02-CASTOR": 2, "03-RUBIS": 3, "04-SAPHIR": 4,
	"05-BABETTE": 5, "REGULATION": 6}

type PegassTime time.Time

func (p *PegassTime) UnmarshalJSON(b []byte) error {
	value := strings.Trim(string(b), `"`)
	if value == "" {
		return nil
	}

	const layout = "2006-01-02T15:04:05"
	t, err := time.Parse(layout, value)
	if err != nil {
		return err
	}
	*p = PegassTime(t)
	return nil
}

func (p *PegassTime) PrintTimePart() string {
	t := time.Time(*p)
	return t.Format("15:04")

}
