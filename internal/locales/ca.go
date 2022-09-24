package locales

import tb "gopkg.in/telebot.v3"

var ca = Locale{
	StartMessage:                        "Si us plau /login per autoritzar Racó Bot.",
	Authorized:                          "Autoritzat",
	AuthorizedResponseMessage:           "Ja pots tancar aquesta pestanya del navegador i tornar a Telegram.",
	GreetingMessage:                     "Hola, %s!",
	HelpMessage:                         "Pots fer servir:\n/test per obtenir una vista prèvia de l'últim avís.\n/logout per deixar de rebre missatges i revocar l'autorització en el servidor.",
	AlreadyLoggedInMessage:              "Ja has iniciat la sessió, comprova /whoami; o /logout per revocar l'autorització.",
	LogoutSucceededMessage:              "Has tancat la sessió amb èxit! I el token de FIB API ha estat revocat al servidor.",
	LogoutFailedMessage:                 "<i>S'ha produït un error en tancar la sessió. Si us plau, intenta-ho de nou més tard.</i>",
	NoticeMessageOriginalLinkText:       "Enllaç",
	NoticeMessageAttachmentNounSingular: "adjunt",
	NoticeMessageAttachmentNounPlural:   "adjunts",
	NoticeMessageAttachmentListHeader:   "📎 <i>Amb %d %s:</i>",
	NoticeMessageTooLongErrorMessage:    "🤖 <i>Ho sento, però aquest missatge és massa llarg per enviar-lo per Telegram, si us plau veges-lo a través <a href=\"%s\">d'aquest enllaç</a>.</i>",
	NoticeUnavailableErrorMessage:       "<i>Avís no disponible</i>",
	NoAvailableNoticesErrorMessage:      "<i>No hi ha avisos disponibles</i>",
	InternalErrorMessage:                "🤖 <i>S'ha produït un error intern</i>",
	FIBAPIAuthorizationExpiredMessage:   "L'autorització ha caducat, si us plau /login per iniciar la sessió de nou.",
	LoginLinkMessage:                    "<a href=\"%s\">Autoritzar Racó Bot</a>",
	SelectPreferredLanguageMenuText:     "Selecciona l'idioma que prefereixis:",
	LanguageUnavailableErrorMessage:     "<i>Idioma no disponible</i>",
	PreferredLanguageSetMessage:         "El teu idioma preferit s'ha configurat a català.",
	DecimalSeparator:                    ',',
	CommandsMenu: []tb.Command{
		{Text: "login", Description: "Autoritzar bot a l'API de la FIB"},
		{Text: "lang", Description: "Seleccionar l'idioma preferit"},
		{Text: "whoami", Description: "Mostrar informació personal"},
		{Text: "test", Description: "Mostrar el darrer avís"},
		{Text: "logout", Description: "Desautoritzar bot"},
	},
}
