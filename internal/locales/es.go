package locales

import tb "gopkg.in/telebot.v3"

var es = Locale{
	StartMessage:                        "Por favor /login para autorizar Racó Bot.",
	Authorized:                          "Autorizado",
	AuthorizedResponseMessage:           "Ya puedes cerrar esta pestaña del navegador y volver a Telegram.",
	GreetingMessage:                     "¡Hola, %s!",
	HelpMessage:                         "Puedes usar:\n/test para obtener una vista previa del último aviso.\n/logout para dejar de recibir mensajes y revocar la autorización en el servidor.",
	AlreadyLoggedInMessage:              "Ya has iniciado la sesión, comprueba /whoami; o /logout para revocar la autorización.",
	LogoutSucceededMessage:              "¡Has cerrado la sesión con éxito! Y el token de FIB API ha sido revocado en el servidor.",
	LogoutFailedMessage:                 "<i>Se ha producido un error al cerrar la sesión. Por favor, intentalo de nuevo más tarde.</i>",
	NoticeMessageOriginalLinkText:       "Enlace",
	NoticeMessageAttachmentNounSingular: "adjunto",
	NoticeMessageAttachmentNounPlural:   "adjuntos",
	NoticeMessageAttachmentListHeader:   "📎 <i>Con %d %s:</i>",
	NoticeMessageTooLongErrorMessage:    "🤖 <i>Lo siento, pero este mensaje es demasiado largo para enviarlo por Telegram, por favor véalo a través de <a href=\"%s\">este enlace</a>.</i>",
	NoticeUnavailableErrorMessage:       "<i>Aviso no disponible</i>",
	NoAvailableNoticesErrorMessage:      "<i>No hay avisos disponibles</i>",
	InternalErrorMessage:                "🤖 <i>Se ha producido un error interno</i>",
	FIBAPIAuthorizationExpiredMessage:   "La autorización ha caducado, por favor /login para iniciar la sesión de nuevo.",
	LoginLinkMessage:                    "<a href=\"%s\">Autorizar Racó Bot</a>",
	SelectPreferredLanguageMenuText:     "Selecciona el idioma que prefieras:",
	LanguageUnavailableErrorMessage:     "<i>Idioma no disponible</i>",
	PreferredLanguageSetMessage:         "Tu idioma preferido se ha configurado a castellano.",
	DecimalSeparator:                    ',',
	CommandsMenu: []tb.Command{
		{Text: "login", Description: "Autorizar bot en la FIB API"},
		{Text: "lang", Description: "Seleccionar el idioma preferido"},
		{Text: "whoami", Description: "Mostrar información personal"},
		{Text: "test", Description: "Mostrar el último aviso"},
		{Text: "logout", Description: "Desautorizar bot"},
	},
}
