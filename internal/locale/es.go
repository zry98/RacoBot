package locale

import tb "gopkg.in/telebot.v3"

var es = Locale{
	StartMessage:                        "Por favor, /login para autorizar Racó Bot.",
	LoginLinkMessage:                    `<a href="%s">Autorizar Racó Bot con UPC SSO.</a>`,
	GreetingMessage:                     "¡Hola, %s!",
	HelpMessage:                         "Puedes usar:\n/test para obtener una vista previa del último aviso.\n/logout para dejar de recibir mensajes y revocar la autorización en el servidor.\n\nPara informes de bugs (avisos con texto mal formado, falta de avisos, error en las traducciones, ...), solicitudes de funciones o cualquier otra consulta, utiliza <i><a href=\"https://github.com/zry98/RacoBot/issues\">GitHub Issues</a></i>, ¡gracias!",
	AlreadyLoggedInMessage:              "Ya has iniciado la sesión, comprueba /whoami; o /logout para revocar la autorización.",
	LogoutSucceededMessage:              "¡Has cerrado la sesión con éxito! Y tu token de FIB API ha sido revocado en el servidor, puedes usar /login para volver a autorizar.",
	LogoutFailedMessage:                 `Se ha producido un error al cerrar la sesión. Aunque el bot ya te eliminó de la base de datos, puedes revocar el token manualmente en <a href="https://api.fib.upc.edu/v2/o/authorized_tokens/">el FIB API Dashboard</a> si lo deseas.`,
	NoticeMessageOriginalLinkText:       "Enlace",
	NoticeMessageAttachmentNounSingular: "adjunto",
	NoticeMessageAttachmentNounPlural:   "adjuntos",
	NoticeMessageAttachmentListHeader:   "<i>📎 Con %d %s:</i>",
	DecimalSeparator:                    ',',
	NoticeMessageTooLongErrorMessage:    `🤖 Lo siento, pero este mensaje es demasiado largo para enviarlo por Telegram, por favor véalo a través de <a href="%s">este enlace</a>.`,
	NoticeUnavailableErrorMessage:       "<i>Aviso no disponible.</i>",
	NoAvailableNoticesErrorMessage:      "<i>No hay avisos disponibles.</i>",
	InternalErrorMessage:                "<i>Se ha producido un error interno.</i>",
	FIBAPIAuthorizationExpiredMessage:   "Tu autorización de <i>FIB API</i> ha caducado, por favor, /login para iniciar la sesión de nuevo.",
	SelectPreferredLanguageMenuText:     "Selecciona el idioma que prefieras:",
	LanguageUnavailableErrorMessage:     "<i>Idioma no disponible.</i>",
	PreferredLanguageSetMessage:         "Tu idioma preferido se ha configurado a castellano.",
	//Authorized:                          "Autorizado",
	//AuthorizedResponseMessage:           "Ya puedes cerrar esta pestaña del navegador y volver a Telegram.",
	CommandsMenu: []tb.Command{
		{Text: "help", Description: "Mostrar el mensaje de ayuda"},
		{Text: "login", Description: "Autorizar bot en la FIB API"},
		{Text: "lang", Description: "Seleccionar el idioma preferido"},
		{Text: "whoami", Description: "Mostrar información personal"},
		{Text: "test", Description: "Mostrar el último aviso"},
		{Text: "logout", Description: "Desautorizar bot"},
	},
}
