package main

import (
	"bytes"
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

func (b *BotService) SendActivitySummary(recipient types.JID, kind ActivityKind, dayCount int) {
	var kindName string
	if kind == SAMU {
		kindName = "SAMU"
	} else {
		kindName = "BSPP"
	}

	err := b.chatClient.SendMessage(fmt.Sprintf("🤖 C'est reçu. Je génère l'état des postes %s sur %d jours.", kindName, dayCount), recipient)
	if err != nil {
		log.Errorf("failed to send whatsapp message: %s", err.Error())
		return
	}

	for i := 0; i < dayCount; i++ {
		var buf = new(bytes.Buffer)

		day := time.Now().AddDate(0, 0, i).Format("2006-01-02")
		log.Infof("fetching activity summary for day '%s' and kind '%#v'", day, kind)
		summary, err := b.pegassClient.GetActivityOnDay(day, kind, false)
		if err != nil {
			log.Errorf("failed to generate activity summary for day '%s' and kind '%d'. error='%s'", day, kind, err.Error())
			err := b.chatClient.SendMessage("Une erreur s'est produite lors de la génération de l'état du réseau. Veuillez réessayer plus tard", recipient)
			if err != nil {
				log.Errorf("failed to notify user that their request could not be processed. Error:'%s'", err.Error())
			}
			return
		}

		if summary == "" {
			summary = "Aucune activité trouvée"
		}

		switch i {
		case 0:
			buf.WriteString(fmt.Sprintf("*Aujourd'hui* (%s) :\n", day))
		case 1:
			buf.WriteString(fmt.Sprintf("*Demain* (%s) :\n", day))
		case 2:
			buf.WriteString(fmt.Sprintf("*Après-demain* (%s) :\n", day))
		}

		buf.WriteString(summary + "\n")
		err = b.chatClient.SendMessage(
			buf.String(),
			recipient,
		)
	}

}
