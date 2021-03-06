package main

import (
	"encoding/json"
	"gopkg.in/urfave/cli.v1"
	"log"
	"os"
)

var pegassClient = PegassClient{}

func main() {
	app := cli.NewApp()
	app.Name = "Pegass CLI"
	app.Usage = "Interact with Red Cross's Pegass web app through the CLI"

	app.Commands = []cli.Command{
		{
			Name:  "login",
			Usage: "Authenticate to Pegass",
			Action: func(c *cli.Context) error {
				configData := parseConfig()
				err := pegassClient.Authenticate(configData.Username, configData.Password)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			Name:  "whoami",
			Usage: "Get current user information",
			Action: func(c *cli.Context) error {
				err := pegassClient.ReAuthenticate()
				if err != nil {
					return err
				}
				user, err := pegassClient.GetCurrentUser()
				if err != nil {
					return err
				}
				log.Printf("Bonjour %s %s (NIVOL: %s) !", user.Utilisateur.Prenom, user.Utilisateur.Nom, user.Utilisateur.ID)
				return nil
			},
		},
		{
			Name:  "dispatchers",
			Usage: "Get list of current dispatchers",
			Action: func(c *cli.Context) error {
				err := pegassClient.ReAuthenticate()
				if err != nil {
					return err
				}

				_, err = pegassClient.GetDispatchers()
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			Name:  "dispatcherstats",
			Usage: "Get dispatcher stats",
			Action: func(c *cli.Context) error {
				err := pegassClient.ReAuthenticate()
				if err != nil {
					return err
				}

				dispatchers, err := pegassClient.GetDispatchers()
				if err != nil {
					return err
				}

				for _, dispatcher := range dispatchers {
					stats, err := pegassClient.GetStatsForUser(dispatcher.ID)
					if err != nil {
						return err
					}

					reguleCount := 0

					for _, statistique := range stats.Statistiques {
						if statistique.StatistiquesGroupeAction.Label == "Urgence et Secourisme" {
							for _, activite := range statistique.StatistiquesActivites {
								if activite.Label == "Régulation" {
									reguleCount = activite.Nombre
									break
								}
							}

							break
						}
					}

					log.Printf("Utilisateur: %s %s : %d régulations", dispatcher.Nom, dispatcher.Prenom, reguleCount)
				}

				return nil
			},
		},
	}
	app.Run(os.Args)
}

func parseConfig() Config {
	configFile, err := os.Open("config.json")
	if err != nil {
		log.Fatal("Failed to open application configuration file 'config.json'", err)
	}
	defer configFile.Close()

	var configData = Config{}
	err = json.NewDecoder(configFile).Decode(&configData)
	if err != nil {
		log.Fatal("Failed to parse application configuration file", err)
	}
	return configData
}
