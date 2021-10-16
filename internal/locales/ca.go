package locales

import tb "gopkg.in/tucnak/telebot.v3"

var ca = Locale{
	StartMessage:                           "Si us plau /login per autoritzar Racó Bot",
	AuthorizedResponseBody:                 "<!DOCTYPE html><html lang=\"ca\"><head><meta charset=\"UTF-8\"><title>Racó Bot</title></head><body><h1>Autoritzat</h1><p>Pot tancar el navegador i tornar a Telegram.</p></body></html>",
	GreetingMessage:                        "Hola, %s!",
	HelpMessage:                            "Feu servir:\n/test per obtenir una vista prèvia de l'últim avís\n/logout per deixar de rebre missatges i revocar l'autorització en el servidor.",
	AlreadyLoggedInMessage:                 "Ja heu iniciat la sessió, comproveu /whoami; o /logout per revocar l'autorització.",
	LogoutSucceededMessage:                 "Heu tancat la sessió amb èxit. I el token de FIB API s'ha revocat al servidor.",
	NoticeMessageAttachmentNounSingular:    "adjunt",
	NoticeMessageAttachmentNounPlural:      "adjunts",
	NoticeMessageAttachmentIndicator:       "%s\n\n<i>- Amb %d %s:</i>\n%s",
	NoticeMessageTooLongErrorMessage:       "[%s] <b>%s</b>\n\nHo sento, però aquest missatge és massa llarg per enviar-lo per Telegram, si us plau consulteu-lo a través <a href=\"%s\">d'aquest enllaç</a>.",
	NoticeUnavailableErrorMessage:          "<i>Avís no disponible</i>",
	NoAvailableNoticesErrorMessage:         "<i>No hi ha avisos disponibles</i>",
	InternalErrorMessage:                   "<i>Error intern</i>",
	FIBAPIAuthorizationExpiredErrorMessage: "L'autorització ha caducat, si us plau /login per iniciar la sessió de nou.",
	LoginLinkMessage:                       "<a href=\"%s\">Autoritzar Racó Bot</a>",
	ChoosePreferredLanguageMenuText:        "Escolliu l'idioma que preferiu:",
	LanguageUnavailableErrorMessage:        "<i>Idioma no disponible</i>",
	PreferredLanguageSetMessage:            "L'idioma preferit s'ha configurat a català.",
	DecimalSeparator:                       ',',
	CommandsMenu: []tb.Command{
		{Text: "login", Description: "Autoritzar bot a l'API de la FIB"},
		{Text: "lang", Description: "Seleccionar l'idioma preferit"},
		{Text: "whoami", Description: "Consultar informació personal"},
		{Text: "test", Description: "Mostra l'últim avís"},
		{Text: "logout", Description: "Desautoritzar bot"},
	},
}
