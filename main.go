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
	app.Name = "Hello_Cli"
	app.Usage = "Print hello world"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "name, n",
			Value: "World",
			Usage: "Who to say hello to.",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "login",
			Usage: "Authenticate to Pegass",
			Action: func(c *cli.Context) error {
				configData := parseConfig()
				pegassClient.Authenticate(configData.Username, configData.Password)
				return nil
			},
		},
		{
			Name: "whoami",
			Usage: "Get current user information",
			Action: func(c *cli.Context) error {
				pegassClient.ReAuthenticate()
				pegassClient.GetCurrentUser()
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
