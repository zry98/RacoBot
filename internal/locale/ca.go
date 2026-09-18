package locale

import tb "gopkg.in/telebot.v3"

var ca = Locale{
	StartMessage:                        "Si us plau, /login per autoritzar Racó Bot.",
	LoginLinkMessage:                    `<a href="%s">Autoritzar Racó Bot amb UPC SSO.</a>`,
	GreetingMessage:                     "Hola, %s!",
	HelpMessage:                         "Pots fer servir:\n/test per obtenir una previsualització de l'últim avís.\n/logout per deixar de rebre els missatges i revocar l'autorització en el servidor.\n\nPer a informes de bugs (avisos amb text mal format, manca d'avisos, error en les traduccions, ...), sol·licituds de noves funcions o qualsevol altra consulta, utilitza <i><a href=\"https://github.com/zry98/RacoBot/issues\">GitHub Issues</a></i>, merci!",
	AlreadyLoggedInMessage:              "Ja has iniciat la sessió, comprova /whoami; o /logout per revocar l'autorització.",
	LogoutSucceededMessage:              "Has tancat la sessió amb èxit! I el token de FIB API ha estat revocat al servidor, pots fer servir /login per tornar a autoritzar.",
	LogoutFailedMessage:                 `S'ha produït un error en tancar la sessió. Encara que el bot ja t'ha eliminat de la base de dades, pots revocar el token manualment a <a href="https://api.fib.upc.edu/v2/o/authorized_tokens/">el FIB API Dashboard</a> si ho desitges.`,
	NoticeMessageOriginalLinkText:       "Enllaç",
	NoticeMessageAttachmentNounSingular: "adjunt",
	NoticeMessageAttachmentNounPlural:   "adjunts",
	NoticeMessageAttachmentListHeader:   "<i>📎 Amb %d %s:</i>",
	DecimalSeparator:                    ',',
	NoticeMessageTooLongErrorMessage:    `🤖 Ho sento, però aquest missatge és massa llarg per enviar-lo per Telegram. Si us plau, veges-lo a través <a href="%s">d'aquest enllaç</a>.`,
	NoticeUnavailableErrorMessage:       "<i>Avís no disponible.</i>",
	NoAvailableNoticesErrorMessage:      "<i>No hi ha avisos disponibles.</i>",
	InternalErrorMessage:                "<i>S'ha produït un error intern.</i>",
	FIBAPIAuthorizationExpiredMessage:   "La teva autorització de <i>FIB API</i> ha caducat, si us plau, /login per iniciar la sessió de nou.",
	SelectPreferredLanguageMenuText:     "Selecciona l'idioma que prefereixis:",
	LanguageUnavailableErrorMessage:     "<i>Idioma no disponible.</i>",
	PreferredLanguageSetMessage:         "El teu idioma preferit s'ha establert en català.",
	BannerNoticesMutedMessage:           "Has silenciat els avisos de banner (aquells que no són d'assignatures, per exemple, eleccions), pots reactivar les notificacions amb /toggle_mute_banner_notices.",
	BannerNoticesUnmutedMessage:         "Has activat les notificacions dels avisos de banner (aquells que no són d'assignatures, per exemple, eleccions), pots silenciar-los amb /toggle_mute_banner_notices.",
	//Authorized:                          "Autoritzat",
	//AuthorizedResponseMessage:           "Ja pots tancar aquesta pestanya del navegador i tornar a Telegram.",
	CommandsMenu: []tb.Command{
		{Text: "help", Description: "Mostra el missatge d'ajuda"},
		{Text: "login", Description: "Autoritzar bot a l'API de la FIB"},
		{Text: "lang", Description: "Seleccionar l'idioma preferit"},
		{Text: "toggle_mute_banner_notices", Description: "Alternar el silenci d'avisos de banner"},
		{Text: "whoami", Description: "Mostrar informació personal"},
		{Text: "test", Description: "Mostrar el darrer avís"},
		{Text: "logout", Description: "Desautoritzar bot"},
	},
}
