package redcross

type GestionDesDroits struct {
	Utilisateur Utilisateur `json:"utilisateur"`
}

type RechercheBenevoles struct {
	List    []Utilisateur `json:"content"`
	Page    int           `json:"number"`
	Total   int           `json:"totalPages"`
	Perpage int           `json:"perpage"`
	Last    bool          `json:"last"`
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
	ID          string        `json:"id"`
	Structure   Structure     `json:"structure"`
	Nom         string        `json:"nom"`
	Prenom      string        `json:"prenom"`
	Coordonnees []Coordonnees `json:"coordonnees"`
	Actif       bool          `json:"actif"`
	Mineur      bool          `json:"mineur"`
	Commentaire string        `json:"commentaire,omitempty"`
}

type Coordonnees struct {
	ID            string `json:"id"`
	UtilisateurID string `json:"utilisateurId"`
	MoyenComID    string `json:"moyenComId"`
	Numero        int    `json:"numero"`
	Libelle       string `json:"libelle"`
	Flag          string `json:"flag"`
	Visible       bool   `json:"visible"`
	CanDelete     bool   `json:"canDelete"`
	CanUpdate     bool   `json:"canUpdate"`
}

type RegulationStats struct {
	OPR, Eval, Regul int
}

type UserTraining struct {
	ID        string `json:"id"`
	Formation struct {
		ID        string `json:"id"`
		Code      string `json:"code"`
		Libelle   string `json:"libelle"`
		Recyclage bool   `json:"recyclage"`
	} `json:"formation"`
	DateObtention string `json:"dateObtention"`
	DateRecyclage string `json:"dateRecyclage,omitempty"`
}
