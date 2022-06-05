package whatsapp

import (
	"context"
	"fmt"
	"github.com/mdp/qrterminal/v3"
	log "github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	"os"
)

type WhatsAppClient struct {
}

func NewClient() WhatsAppClient {
	return WhatsAppClient{}
}

func (whatsappClient *WhatsAppClient) RegisterDevice() error {
	client, err := initClient()
	if err != nil {
		return nil
	}

	if client.Store.ID == nil {
		qrChannel, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			return err
		}
		for evt := range qrChannel {
			if evt.Event == "code" {
				fmt.Println("Please scan the following QRCode with your What's App client, in order to link the device")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		return fmt.Errorf("Device is already registered to a What's App account: %s", client.Store.PushName)
	}

	return nil
}

func (whatsAppClient *WhatsAppClient) SendMessageToGroup(message string, groupId types.JID) error {
	client, err := initClient()
	if err != nil {
		return err
	}

	err = client.Connect()
	if err != nil {
		return err
	}

	_, err = client.SendMessage(groupId, "", &waProto.Message{Conversation: proto.String(message)})
	if err != nil {
		return err
	}
	return nil
}

func initClient() (*whatsmeow.Client, error) {
	var minLogLevel = "INFO"
	if os.Getenv("VERBOSE") != "" {
		minLogLevel = "DEBUG"
	}
	dbLog := waLog.Stdout("Database", minLogLevel, true)
	container, err := sqlstore.New("sqlite3", "file:pegass.db?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, err
	}
	device, err := container.GetFirstDevice()
	if err != nil {
		return nil, err
	}
	clientLog := waLog.Stdout("Client", minLogLevel, true)
	return whatsmeow.NewClient(device, clientLog), nil
}

func (w *WhatsAppClient) PrintGroupList() error {
	client, err := initClient()
	if err != nil {
		return err
	}
	err = client.Connect()
	if err != nil {
		return err
	}
	groups, err := client.GetJoinedGroups()
	if err != nil {
		return err
	}
	for _, group := range groups {
		log.Infof("Group '%s' - JID: '%s'", group.Name, group.JID)
	}
	return nil
}
