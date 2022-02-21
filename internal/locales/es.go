package locales

import tb "gopkg.in/telebot.v3"

var es = Locale{
	StartMessage:                           "Por favor /login para autorizar Racó Bot.",
	AuthorizedResponseBody:                 "<!DOCTYPE html><html lang=\"es\"><head><meta charset=\"UTF-8\"><title>Racó Bot</title></head><body><h1>Autorizado</h1><p>Puede cerrar el navegador y volver a Telegram.</p></body></html>",
	GreetingMessage:                        "¡Hola, %s!",
	HelpMessage:                            "Puede usar:\n/test para obtener una vista previa del último aviso\n/logout para dejar de recibir mensajes y revocar la autorización en el servidor.",
	AlreadyLoggedInMessage:                 "Ya has iniciado la sesión, compruebe /whoami; o /logout para revocar la autorización.",
	LogoutSucceededMessage:                 "¡Has terminado la sesión con éxito! Y el token de FIB API ha sido revocado en el servidor.",
	NoticeMessageAttachmentNounSingular:    "adjunto",
	NoticeMessageAttachmentNounPlural:      "adjuntos",
	NoticeMessageAttachmentIndicator:       "%s\n\n<i>- Con %d %s:</i>\n%s",
	NoticeMessageTooLongErrorMessage:       "[%s] <b>%s</b>\n\nLo siento, pero este mensaje es demasiado largo para enviarlo por Telegram, por favor véalo a través de <a href=\"%s\">este enlace</a>.",
	NoticeUnavailableErrorMessage:          "<i>Aviso no disponible</i>",
	NoAvailableNoticesErrorMessage:         "<i>No hay avisos disponibles</i>",
	InternalErrorMessage:                   "<i>Error interno</i>",
	FIBAPIAuthorizationExpiredErrorMessage: "La autorización ha caducado, por favor /login para iniciar la sesión de nuevo.",
	LoginLinkMessage:                       "<a href=\"%s\">Autorizar Racó Bot</a>",
	ChoosePreferredLanguageMenuText:        "Elija su idioma preferido:",
	LanguageUnavailableErrorMessage:        "<i>Idioma no disponible</i>",
	PreferredLanguageSetMessage:            "El idioma preferido se ha configurado a castellano.",
	DecimalSeparator:                       ',',
	CommandsMenu: []tb.Command{
		{Text: "login", Description: "Autorizar bot en la API de la FIB"},
		{Text: "lang", Description: "Seleccionar el idioma preferido"},
		{Text: "whoami", Description: "Consultar información personal"},
		{Text: "test", Description: "Mostrar el último aviso"},
		{Text: "logout", Description: "Desautorizar bot"},
	},
}
