package redcross

type GestionDesDroits struct {
	Utilisateur Utilisateur `json:"utilisateur"`
}

type RechercheBenevoles struct {
	List     []Utilisateur `json:"list"`
	Page     int           `json:"page"`
	Pages    int           `json:"pages"`
	Total    int           `json:"total"`
	CanAdmin bool          `json:"canAdmin"`
	Perpage  int           `json:"perpage"`
}

type Structure struct {
	ID            int    `json:"id"`
	TypeStructure string `json:"typeStructure"`
	Libelle       string `json:"libelle"`
	LibelleCourt  string `json:"libelleCourt"`
	Adresse       string `json:"adresse"`
	Telephone     string `json:"telephone"`
	Mail          string `json:"mail"`
	SiteWeb       string `json:"siteWeb"`
	Parent        struct {
		ID int `json:"id"`
	} `json:"parent"`
	StructureMenantActiviteList []struct {
		ID      int    `json:"id"`
		Libelle string `json:"libelle"`
	} `json:"structureMenantActiviteList"`
}

type Utilisateur struct {
	ID          string    `json:"id"`
	Structure   Structure `json:"structure"`
	Nom         string    `json:"nom"`
	Prenom      string    `json:"prenom"`
	Coordonnees []struct {
		ID struct {
			UtilisateurID string `json:"utilisateurId"`
			MoyenComID    string `json:"moyenComId"`
			Numero        int    `json:"numero"`
		} `json:"id"`
		UtilisateurID string `json:"utilisateurId"`
		MoyenComID    string `json:"moyenComId"`
		Numero        int    `json:"numero"`
		Libelle       string `json:"libelle"`
		Flag          string `json:"flag"`
		Visible       bool   `json:"visible"`
		CanDelete     bool   `json:"canDelete"`
		CanUpdate     bool   `json:"canUpdate"`
	} `json:"coordonnees"`
	Actif       bool   `json:"actif"`
	Mineur      bool   `json:"mineur"`
	Commentaire string `json:"commentaire,omitempty"`
}
