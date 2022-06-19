package main

import (
	"fmt"
	"github.com/fabien-chebel/pegass-cli/whatsapp"
	log "github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/types"
	"time"
)

type BotService struct {
	pegassClient *PegassClient
	chatClient   *whatsapp.WhatsAppClient
}

func (b *BotService) SendActivitySummary(recipient types.JID, kind ActivityKind) {

	err := b.chatClient.SendMessage("🤖 C'est reçu. Je génère l'état des postes de demain.", recipient)
	if err != nil {
		log.Errorf("failed to send whatsapp message: %s", err.Error())
		return
	}

	day := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	log.Info("Fetching activity summary for day ", day)
	summary, err := b.pegassClient.GetActivityOnDay(day, kind)
	if err != nil {
		log.Errorf("failed to generate activity summary for day '%s' and kind '%d'", day, kind)
		return
	}
	summary = fmt.Sprintf("Etat du réseau de secours de demain (%s):\n%s", day, summary)

	err = b.chatClient.SendMessage(
		summary,
		recipient,
	)
}
