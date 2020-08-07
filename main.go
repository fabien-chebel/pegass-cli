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
				err = pegassClient.GetCurrentUser()
				if err != nil {
					return err
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
