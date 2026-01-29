package services

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var Bot *tgbotapi.BotAPI
var AdminChatID int64 // Este lo guardaremos cuando nos escribas /start

func InitBot(token string) error {
	var err error
	Bot, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		return err
	}

	log.Printf("Bot autorizado en la cuenta %s", Bot.Self.UserName)
	
	// Correr un listener en segundo plano para captar el Admin ID
	go listenForCommands()
	
	return nil
}

func listenForCommands() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := Bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				AdminChatID = update.Message.Chat.ID
				msg := tgbotapi.NewMessage(AdminChatID, fmt.Sprintf("¡Hola Admin! Tu ID ha sido registrado: %d. Ahora recibirás notificaciones aquí.", AdminChatID))
				Bot.Send(msg)
				log.Printf("Admin Chat ID registrado: %d", AdminChatID)
			}
		}
	}
}

func NotifyAdmin(text string) {
	if Bot == nil || AdminChatID == 0 {
		log.Println("Bot no iniciado o AdminChatID desconocido")
		return
	}

	msg := tgbotapi.NewMessage(AdminChatID, text)
	_, err := Bot.Send(msg)
	if err != nil {
		log.Printf("Error enviando notificación: %v", err)
	}
}
