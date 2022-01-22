package redcross

type Competences []struct {
	ID       int    `json:"id"`
	Libelle  string `json:"libelle"`
	Active   bool   `json:"active"`
	ReadOnly bool   `json:"readOnly"`
}

type Role struct {
	ID      string `json:"id"`
	Libelle string `json:"libelle"`
	Type    string `json:"type"`
}
