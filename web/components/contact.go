package components

import (
	"net/url"
	"strings"
)

const (
	WhatsAppPhoneDisplay    = "+55 27 99859-5125"
	WhatsAppNumberCanonical = "5527998595125"
	WhatsAppBaseURL         = "https://wa.me/" + WhatsAppNumberCanonical
	WhatsAppGeneralMessage  = "Olá! Vim pelo site da PrintLab 👋 Quero saber mais sobre um produto ou impressão 3D personalizada. Pode me ajudar?"
	WhatsAppCatalogMessage  = "Olá! Vim pelo catálogo da PrintLab e gostaria de tirar uma dúvida sobre impressão 3D. Pode me ajudar?"
)

func WhatsAppURL(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		message = WhatsAppGeneralMessage
	}
	query := url.Values{}
	query.Set("text", message)
	return WhatsAppBaseURL + "?" + query.Encode()
}

func WhatsAppGeneralURL() string {
	return WhatsAppURL(WhatsAppGeneralMessage)
}

func WhatsAppCatalogURL() string {
	return WhatsAppURL(WhatsAppCatalogMessage)
}
