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
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type WhatsAppClient struct {
	client            *whatsmeow.Client
	onMessageReceived MessageCallback
}

func NewClient() WhatsAppClient {
	return WhatsAppClient{}
}

func (w *WhatsAppClient) RegisterDevice() error {
	err := w.initClient()
	if err != nil {
		return err
	}

	if w.client.Store.ID == nil {
		qrChannel, _ := w.client.GetQRChannel(context.Background())
		err = w.client.Connect()
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
		return fmt.Errorf("Device is already registered to a What's App account: %s", w.client.Store.PushName)
	}

	return nil
}

func (w *WhatsAppClient) SendMessage(message string, groupId types.JID) error {
	err := w.initAndConnectIfNecessary()
	if err != nil {
		return err
	}

	_, err = w.client.SendMessage(context.Background(), groupId, &waProto.Message{Conversation: proto.String(message)})
	if err != nil {
		return err
	}
	return nil
}

func (w *WhatsAppClient) initClient() error {
	var minLogLevel = "INFO"
	if os.Getenv("VERBOSE") != "" {
		minLogLevel = "DEBUG"
	}
	dbLog := waLog.Stdout("Database", minLogLevel, true)
	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite", "file:pegass.db?_pragma=foreign_keys(1)", dbLog)
	if err != nil {
		return err
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return err
	}
	clientLog := waLog.Stdout("Client", minLogLevel, true)
	w.client = whatsmeow.NewClient(device, clientLog)
	return nil
}

func (w *WhatsAppClient) initAndConnectIfNecessary() error {
	if w.client == nil {
		err := w.initClient()
		if err != nil {
			return err
		}
	}

	if !w.client.IsConnected() {
		err := w.client.Connect()
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *WhatsAppClient) PrintGroupList() error {
	err := w.initAndConnectIfNecessary()
	if err != nil {
		return err
	}
	groups, err := w.client.GetJoinedGroups(context.Background())
	if err != nil {
		return err
	}
	for _, group := range groups {
		log.Infof("Group '%s' - JID: '%s'", group.Name, group.JID)
	}
	return nil
}

func (w *WhatsAppClient) StartBot() error {
	err := w.initAndConnectIfNecessary()
	if err != nil {
		return err
	}

	w.client.AddEventHandler(w.eventHandler)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	w.client.Disconnect()
	return nil
}

func (w *WhatsAppClient) SetMessageCallback(callback MessageCallback) {
	w.onMessageReceived = callback
}

func (w *WhatsAppClient) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		if w.onMessageReceived != nil {
			w.onMessageReceived(v.Info.PushName, v.Info.Sender, v.Info.Chat, v.Message.GetConversation(), v.Info.Timestamp)
		}
	case *events.Disconnected:
		log.Warn("WhatsApp client was disconnected")
	case *events.ConnectFailure:
		log.Warnf("failed to connect to whatsapp: %#v", v.Reason)
	case *events.LoggedOut:
		log.Warnf("Received 'loged out' event: %#v", v.Reason)
	case *events.StreamReplaced:
		log.Warnf("Another WhatsApp client connected and stole our session! Will reconnect in a few seconds")
		w.client.Disconnect()
		time.Sleep(5 * time.Second)
		err := w.initAndConnectIfNecessary()
		if err != nil {
			log.Error("failed to reconnect to whatsapp", err)
		}
	}
}

type MessageCallback func(senderName string, senderId types.JID, chatId types.JID, content string, timestamp time.Time)
