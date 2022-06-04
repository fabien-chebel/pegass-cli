package redcross

type AdvancedSearch struct {
	StructureList   []int    `json:"structureList"`
	FormationInList []string `json:"formationInList"`
	SearchType      string   `json:"searchType"`
	WithMoyensCom   bool     `json:"withMoyensCom"`
}

type StructureList struct {
	ID               string `json:"id"`
	Nom              string `json:"nom"`
	TypeZoneGeo      string `json:"typeZoneGeo"`
	StructuresFilles []struct {
		ID            int    `json:"id"`
		TypeStructure string `json:"typeStructure"`
		Libelle       string `json:"libelle"`
	} `json:"structuresFilles"`
}
