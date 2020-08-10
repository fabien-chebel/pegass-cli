package redcross

type StatsBenevole struct {
	Debut        string `json:"debut"`
	Fin          string `json:"fin"`
	Unite        string `json:"unite"`
	Statistiques []struct {
		StatistiquesGroupeAction Stats   `json:"statistiquesGroupeAction"`
		StatistiquesActivites    []Stats `json:"statistiquesActivites"`
	} `json:"statistiques"`
	StatistiquesActivite []Stats `json:"statistiquesActivite"`
	IDUtilisateur        string  `json:"idUtilisateur"`
}
type Stats struct {
	ID          interface{} `json:"id"`
	Label       string      `json:"label"`
	Tri         interface{} `json:"tri"`
	Nombre      int         `json:"nombre"`
	Pourcentage float64     `json:"pourcentage"`
}
