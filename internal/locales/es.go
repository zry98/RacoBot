package locales

var es = Locale{
	StartMessage:                           "Por favor /login para autorizar Racó Bot.",
	GreetingMessage:                        "¡Hola, %s!",
	HelpMessage:                            "Puede usar:\n/test para obtener una vista previa del último aviso\n/logout para dejar de recibir mensajes y revocar la autorización en el servidor.",
	AlreadyLoggedInMessage:                 "Ya has iniciado la sesión, compruebe /whoami; o /logout para revocar la autorización.",
	LogoutSucceededMessage:                 "¡Has terminado la sesión con éxito! Y el token de FIB API ha sido revocado en el servidor.",
	NoticeMessageAttachmentNounSingular:    "adjunto",
	NoticeMessageAttachmentNounPlural:      "adjuntos",
	NoticeMessageAttachmentIndicator:       "%s\n\n<i>- Con %d %s:</i>\n%s",
	NoticeMessageTooLongErrorMessage:       "[%s] <b>%s</b>\n\nLo siento, pero este mensaje es demasiado largo para enviarlo por Telegram, por favor véalo a través de <a href=\"%s\">este enlace</a>.",
	NoticeUnavailableErrorMessage:          "<i>Aviso no disponible</i>",
	NoNoticesAvailableErrorMessage:         "<i>No hay avisos disponibles</i>",
	InternalErrorMessage:                   "<i>Error interno</i>",
	FIBAPIAuthorizationExpiredErrorMessage: "La autorización ha caducado, por favor /login para iniciar la sesión de nuevo.",
	LoginLinkMessage:                       "<a href=\"%s\">Autorizar Racó Bot</a>",
}
