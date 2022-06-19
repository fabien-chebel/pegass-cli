package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/fabien-chebel/pegass-cli/whatsapp"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/types"
	"gopkg.in/urfave/cli.v1"
	"os"
	"strconv"
	"strings"
	"time"
)

var pegassClient PegassClient

func initClient() (Config, error) {
	configData := parseConfig()
	pegassClient = PegassClient{
		Username:      configData.Username,
		Password:      configData.Password,
		TotpSecretKey: configData.TotpSecretKey,
	}
	return configData, pegassClient.Authenticate()
}

func initLogs(verbose bool) {
	log.SetOutput(os.Stdout)
	if verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func main() {
	initLogs(os.Getenv("VERBOSE") != "")

	app := cli.NewApp()
	app.Name = "Pegass CLI"
	app.Usage = "Interact with Red Cross's Pegass web app through the CLI"

	app.Commands = []cli.Command{
		{
			Name:  "login",
			Usage: "Authenticate to Pegass",
			Action: func(c *cli.Context) error {
				_, err := initClient()
				if err != nil {
					return nil
				}
				err = pegassClient.Authenticate()
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
				_, err := initClient()
				if err != nil {
					return err
				}
				user, err := pegassClient.GetCurrentUser()
				if err != nil {
					return err
				}
				log.Infof("Bonjour %s %s (NIVOL: %s) !", user.Utilisateur.Prenom, user.Utilisateur.Nom, user.Utilisateur.ID)
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

					log.Infof("Utilisateur: %s %s : %d régulations", dispatcher.Nom, dispatcher.Prenom, reguleCount)
				}

				return nil
			},
		},
		{
			Name:  "regulationstats",
			Usage: "Export regulation stats",
			Action: func(c *cli.Context) error {
				err := pegassClient.ReAuthenticate()
				if err != nil {
					return err
				}

				statsByUser, err := pegassClient.GetActivityStats()
				if err != nil {
					return err
				}

				f, err := os.Create("stats-regulation.csv")
				defer f.Close()
				if err != nil {
					return err
				}

				w := csv.NewWriter(f)
				defer w.Flush()

				err = w.Write([]string{"nom,prenom,regule,eval,opr"})
				if err != nil {
					return err
				}

				for nivol, stats := range statsByUser {
					details, err := pegassClient.GetUserDetails(nivol)
					if err != nil {
						log.Printf("failed to fetch user details for user '%s' ; %s", nivol, err)
					}
					log.Printf("Utilisateur %s %s ; %d regulations, %d eval, %d OPR", details.Nom, details.Prenom, stats.Regul, stats.Eval, stats.OPR)
					record := []string{details.Nom, details.Prenom, strconv.Itoa(stats.Regul), strconv.Itoa(stats.Eval), strconv.Itoa(stats.OPR)}
					err = w.Write(record)
					if err != nil {
						return err
					}
				}

				return nil
			},
		},
		{
			Name:  "find-users-for-role",
			Usage: "Export a list of users matching a given pegass role",
			Action: func(c *cli.Context) error {
				roleName := c.Args().Get(0)

				err := pegassClient.ReAuthenticate()
				if err != nil {
					return err
				}

				role, err := pegassClient.FindRoleByName(roleName)
				if err != nil {
					return err
				}

				log.Printf("Found role {id: '%s', type: '%s', name: '%s'} for role name '%s'", role.ID, role.Type, role.Libelle, roleName)

				users, err := pegassClient.GetUsersForRole(role)

				f, err := os.Create(fmt.Sprintf("user-export-92-%s-%s.csv", role.Type, role.ID))
				defer f.Close()
				if err != nil {
					return err
				}

				w := csv.NewWriter(f)
				defer w.Flush()

				err = w.Write([]string{"nom", "prenom", "UL", "nivol", "phone-number", "role"})
				if err != nil {
					return err
				}

				for _, user := range users {
					phoneNumber := ""
					for _, coordonnee := range user.Coordonnees {
						if coordonnee.MoyenComID == "POR" {
							phoneNumber = coordonnee.Libelle
							break
						}
					}

					record := []string{user.Nom, user.Prenom, user.Structure.Libelle, user.ID, phoneNumber, roleName}
					err = w.Write(record)
					if err != nil {
						return err
					}
				}

				return nil
			},
		},
		{
			Name:  "summarize-samu-activities",
			Usage: "Fetch tomorrow's SAMU-related activities and send their status to WhatsApp",
			Action: func(c *cli.Context) error {
				conf, err := initClient()
				if err != nil {
					return err
				}

				day := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
				log.Info("Fetching activity summary for day ", day)
				summary, err := pegassClient.GetActivityOnDay(day, SAMU)
				if err != nil {
					return err
				}
				summary = fmt.Sprintf("Etat du réseau de secours de demain (%s):\n%s", day, summary)
				log.Info(summary)

				if conf.WhatsAppNotificationGroup == "" {
					return fmt.Errorf("no WhatsApp group Id provided. Skipping WhatsApp notification")
				}
				whatsAppClient := whatsapp.NewClient()
				jid, err := types.ParseJID(conf.WhatsAppNotificationGroup)
				if err != nil {
					return err
				}
				err = whatsAppClient.SendMessage(
					summary,
					jid,
				)

				return err
			},
		},
		{
			Name:  "register-chat-device",
			Usage: "Register whats app device locally",
			Action: func(c *cli.Context) error {
				whatsAppClient := whatsapp.NewClient()
				err := whatsAppClient.RegisterDevice()
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			Name: "list-chat-groups",
			Action: func(c *cli.Context) error {
				whatsAppClient := whatsapp.NewClient()
				return whatsAppClient.PrintGroupList()
			},
		},
		{
			Name: "start-bot",
			Action: func(c *cli.Context) error {
				_, err := initClient()
				if err != nil {
					return err
				}

				whatsAppClient := whatsapp.NewClient()
				var botService = BotService{
					pegassClient: &pegassClient,
					chatClient:   &whatsAppClient,
				}
				whatsAppClient.SetMessageCallback(func(senderName string, senderId types.JID, chatId types.JID, content string) {
					log.Infof("Received message from '%s': %s", senderName, content)
					var recipient = senderId
					if chatId != (types.JID{}) {
						recipient = chatId
					}
					lowerMessage := strings.ToLower(content)

					if strings.HasPrefix(lowerMessage, "!psr") {
						botService.SendActivitySummary(recipient, SAMU)
					} else if strings.HasPrefix(lowerMessage, "!bspp") {
						botService.SendActivitySummary(recipient, BSPP)
					}
				})
				err = whatsAppClient.StartBot()
				if err != nil {
					return err
				}
				return nil
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
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
