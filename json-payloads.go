package main

type GestionDesDroits struct {
	Utilisateur Utilisateur `json:"utilisateur"`
}

type Utilisateur struct {
	Id string `json:"id"`
	Nom string `json:"nom"`
	Prenom string `json:"prenom"`
	Actif bool `json:"actif"`
	Mineur bool `json:"mineur"`
}
