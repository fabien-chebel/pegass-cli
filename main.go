package main

import (
	"encoding/csv"
	"encoding/json"
	"github.com/fabien-chebel/pegass-cli/redcross"
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
			Name:  "regulation",
			Usage: "Get dispatcher activities",
			Action: func(c *cli.Context) error {
				err := pegassClient.ReAuthenticate()
				if err != nil {
					return err
				}

				var regulations []redcross.Regulation
				regulations, err = pegassClient.GetDispatcherSchedule("2019-01-01", "2019-12-31")
				if err != nil {
					return err
				}

				csvWriter := csv.NewWriter(os.Stdout)
				csvWriter.Write([]string{
					"startDate",
					"endDate",
					"nivolRegulateur",
					"nomRegulateur",
					"prenomRegulateur",
				})

				for _, regulation := range regulations {
					err = csvWriter.Write([]string{
						regulation.Debut.String(),
						regulation.Fin.String(),
						regulation.Regulateur.ID,
						regulation.Regulateur.Nom,
						regulation.Regulateur.Prenom,
					})
					if err != nil {
						log.Fatalln("failed to write csv output", err)
					}
				}
				csvWriter.Flush()

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
