package email

func SignupConfirmation(data LinkEmailData) (Message, error) {
	if err := data.validate(); err != nil {
		return Message{}, err
	}
	return render(TemplateSignupConfirmation, "Confirme seu cadastro na PrintLab", pageData{
		SiteURL:       data.SiteURL,
		Preheader:     "Confirme seu cadastro para acessar sua conta PrintLab.",
		Title:         "Confirme seu cadastro",
		Intro:         greeting(data.RecipientName) + "Use o botão abaixo para confirmar seu cadastro na PrintLab. Se você não solicitou esse acesso, ignore este e-mail.",
		CTAURL:        data.ActionURL,
		CTALabel:      "Confirmar cadastro",
		FallbackLabel: "Se o botão não funcionar, copie e cole este link no navegador",
		FooterNote:    "Este link deve ser usado somente por quem solicitou o cadastro.",
	})
}

func MagicLink(data LinkEmailData) (Message, error) {
	if err := data.validate(); err != nil {
		return Message{}, err
	}
	return render(TemplateMagicLink, "Seu link de acesso à PrintLab", pageData{
		SiteURL:       data.SiteURL,
		Preheader:     "Acesse sua conta PrintLab com este link seguro.",
		Title:         "Acesse sua conta",
		Intro:         greeting(data.RecipientName) + "Use o link abaixo para acessar sua conta PrintLab. O link é pessoal e não deve ser compartilhado.",
		CTAURL:        data.ActionURL,
		CTALabel:      "Acessar PrintLab",
		FallbackLabel: "Se o botão não funcionar, copie e cole este link no navegador",
		FooterNote:    "Se você não pediu este acesso, ignore este e-mail.",
	})
}

func AccessRecovery(data LinkEmailData) (Message, error) {
	if err := data.validate(); err != nil {
		return Message{}, err
	}
	return render(TemplateAccessRecovery, "Recupere seu acesso à PrintLab", pageData{
		SiteURL:       data.SiteURL,
		Preheader:     "Use este link para recuperar o acesso à sua conta PrintLab.",
		Title:         "Recuperação de acesso",
		Intro:         greeting(data.RecipientName) + "Recebemos uma solicitação para recuperar seu acesso à PrintLab. Use o botão abaixo para continuar.",
		CTAURL:        data.ActionURL,
		CTALabel:      "Recuperar acesso",
		FallbackLabel: "Se o botão não funcionar, copie e cole este link no navegador",
		FooterNote:    "Se você não solicitou recuperação de acesso, ignore este e-mail.",
	})
}

func OrderConfirmation(data OrderEmailData) (Message, error) {
	if err := data.validate(false); err != nil {
		return Message{}, err
	}
	return render(TemplateOrderConfirmation, "Recebemos seu pedido "+data.OrderNumberLabel, pageData{
		SiteURL:       data.SiteURL,
		Preheader:     "Resumo comercial do seu pedido PrintLab.",
		Title:         "Pedido recebido",
		Intro:         greeting(data.CustomerName) + "Recebemos seu pedido e preparamos o resumo abaixo com os dados comerciais da compra.",
		CTAURL:        firstURL(data.OrderURL, data.TrackingURL),
		CTALabel:      "Ver pedido",
		FallbackLabel: "Se o botão não funcionar, copie e cole este link no navegador",
		FooterNote:    "Este e-mail usa o estado persistido do pedido e não recota frete nem altera pagamento.",
		Order:         &data,
	})
}

func OrderStatusUpdate(data OrderEmailData) (Message, error) {
	if err := data.validate(true); err != nil {
		return Message{}, err
	}
	return render(TemplateOrderStatusUpdate, "Atualização do pedido "+data.OrderNumberLabel, pageData{
		SiteURL:       data.SiteURL,
		Preheader:     "Seu pedido PrintLab teve uma atualização de status.",
		Title:         "Atualização do pedido",
		Intro:         greeting(data.CustomerName) + "O status do seu pedido foi atualizado. Confira o resumo abaixo.",
		CTAURL:        firstURL(data.TrackingURL, data.OrderURL),
		CTALabel:      "Acompanhar pedido",
		FallbackLabel: "Se o botão não funcionar, copie e cole este link no navegador",
		FooterNote:    "A página de acompanhamento mostra somente informações minimizadas do pedido.",
		Order:         &data,
	})
}

func PaymentConfirmed(data OrderEmailData) (Message, error) {
	data.StatusLabel = defaultString(data.StatusLabel, "Pagamento confirmado")
	if err := data.validate(true); err != nil {
		return Message{}, err
	}
	return render(TemplatePaymentConfirmed, "Pagamento confirmado do pedido "+data.OrderNumberLabel, pageData{
		SiteURL:       data.SiteURL,
		Preheader:     "Pagamento confirmado para seu pedido PrintLab.",
		Title:         "Pagamento confirmado",
		Intro:         greeting(data.CustomerName) + "O pagamento do seu pedido foi confirmado. A PrintLab seguirá com as próximas etapas de produção e entrega.",
		CTAURL:        firstURL(data.TrackingURL, data.OrderURL),
		CTALabel:      "Acompanhar pedido",
		FallbackLabel: "Se o botão não funcionar, copie e cole este link no navegador",
		FooterNote:    "Este e-mail não contém detalhes internos de produção, pagamento ou logística.",
		Order:         &data,
	})
}

func greeting(name string) string {
	if name == "" {
		return ""
	}
	return "Olá, " + name + ". "
}

func firstURL(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
