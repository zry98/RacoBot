package locales

var ca = Locale{
	StartMessage:                           "Si us plau /login per autoritzar Racó Bot",
	GreetingMessage:                        "Hola, %s!",
	HelpMessage:                            "Feu servir:\n/test per obtenir una vista prèvia de l'últim avís\n/logout per deixar de rebre missatges i revocar l'autorització en el servidor.",
	AlreadyLoggedInMessage:                 "Ja heu iniciat la sessió, comproveu /whoami; o /logout per revocar l'autorització.",
	LogoutSucceededMessage:                 "Heu tancat la sessió amb èxit. I el token de FIB API s'ha revocat al servidor.",
	NoticeMessageAttachmentNounSingular:    "adjunt",
	NoticeMessageAttachmentNounPlural:      "adjunts",
	NoticeMessageAttachmentIndicator:       "%s\n\n<i>- Amb %d %s:</i>\n%s",
	NoticeMessageTooLongErrorMessage:       "[%s] <b>%s</b>\n\nHo sento, però aquest missatge és massa llarg per enviar-lo per Telegram, si us plau consulteu-lo a través <a href=\"%s\">d'aquest enllaç</a>.",
	NoticeUnavailableErrorMessage:          "<i>Avís no disponible</i>",
	NoNoticesAvailableErrorMessage:         "<i>No hi ha avisos disponibles</i>",
	InternalErrorMessage:                   "<i>Error intern</i>",
	FIBAPIAuthorizationExpiredErrorMessage: "L'autorització ha caducat, si us plau /login per iniciar la sessió de nou.",
	LoginLinkMessage:                       "<a href=\"%s\">Autoritzar Racó Bot</a>",
}