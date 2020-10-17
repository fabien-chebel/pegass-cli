package redcross

import "time"

type Regulation struct {
	ID         string
	Statut     string
	Debut      time.Time
	Fin        time.Time
	Regulateur Utilisateur
}

type Seance struct {
	ID          string      `json:"id"`
	Debut       string      `json:"debut"`
	Fin         string      `json:"fin"`
	Statut      string      `json:"statut"`
	Utilisateur Utilisateur `json:"utilisateur"`
	Role        string      `json:"role"`
	Type        string      `json:"type"`
}

type Activity struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Libelle      string `json:"libelle"`
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
	Recurrence struct {
		ID                   string `json:"id"`
		CreationEnMasse      bool   `json:"creationEnMasse"`
		Type                 string `json:"type"`
		CreneauRecurrentList []struct {
			ID                       int    `json:"id"`
			NumeroJourDansRecurrence int    `json:"numeroJourDansRecurrence"`
			DateReferenceDebut       string `json:"dateReferenceDebut"`
			DateReferenceFin         string `json:"dateReferenceFin"`
			RoleConfigList           []struct {
				ID       int    `json:"id"`
				Code     string `json:"code"`
				Role     string `json:"role"`
				Actif    bool   `json:"actif"`
				Effectif int    `json:"effectif"`
				Type     string `json:"type"`
			} `json:"roleConfigList"`
		} `json:"creneauRecurrentList"`
		Debut string `json:"debut"`
		Fin   string `json:"fin"`
	} `json:"recurrence"`
	StructureCreateur struct {
		ID int `json:"id"`
	} `json:"structureCreateur"`
	Statut                 string `json:"statut"`
	StructureOrganisatrice struct {
		ID int `json:"id"`
	} `json:"structureOrganisatrice"`
	StructureMenantActivite struct {
		ID int `json:"id"`
	} `json:"structureMenantActivite"`
	SeanceList []struct {
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
		Debut          string `json:"debut"`
		Fin            string `json:"fin"`
		Adresse        string `json:"adresse"`
		RevisionNumber int    `json:"revisionNumber"`
		RoleConfigList []struct {
			ID       string `json:"id"`
			Code     string `json:"code"`
			Role     string `json:"role"`
			Actif    bool   `json:"actif"`
			Effectif int    `json:"effectif"`
			Type     string `json:"type"`
		} `json:"roleConfigList"`
	} `json:"seanceList"`
	Responsable struct {
		ID     string `json:"id"`
		Actif  bool   `json:"actif"`
		Mineur bool   `json:"mineur"`
	} `json:"responsable"`
	Rappel      bool   `json:"rappel"`
	Commentaire string `json:"commentaire,omitempty"`
}
