package redcross

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
	Utilisateur struct {
		ID     string `json:"id"`
		Actif  bool   `json:"actif"`
		Mineur bool   `json:"mineur"`
	} `json:"utilisateur"`
	Debut      string `json:"debut"`
	Fin        string `json:"fin"`
	Statut     string `json:"statut"`
	Role       string `json:"role"`
	Type       string `json:"type"`
	IsMultiple bool   `json:"isMultiple"`
}
