package locales

var en = Locale{
	StartMessage:                           "Please /login to authorize Racó Bot",
	AuthorizedResponseBody:                 "<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"UTF-8\"><title>Racó Bot</title></head><body><h1>Authorized</h1><p>You can close the browser and return to Telegram.</p></body></html>",
	GreetingMessage:                        "Hello, %s!",
	HelpMessage:                            "You can use:\n/test to preview the latest one notice\n/logout to stop receiving messages and revoke the authorization on server.",
	AlreadyLoggedInMessage:                 "Already logged-in, check /whoami; or, /logout to revoke the authorization.",
	LogoutSucceededMessage:                 "You have successfully logged out! And the FIB API token has been revoked on server.",
	NoticeMessageAttachmentNounSingular:    "attachment",
	NoticeMessageAttachmentNounPlural:      "attachments",
	NoticeMessageAttachmentIndicator:       "%s\n\n<i>- With %d %s:</i>\n%s",
	NoticeMessageTooLongErrorMessage:       "[%s] <b>%s</b>\n\nSorry, but this message is too long to be sent through Telegram, please view it through <a href=\"%s\">this link</a>.",
	NoticeUnavailableErrorMessage:          "<i>Notice unavailable</i>",
	NoAvailableNoticesErrorMessage:         "<i>No available notices</i>",
	InternalErrorMessage:                   "<i>Internal error</i>",
	FIBAPIAuthorizationExpiredErrorMessage: "Authorization expired, please /login again.",
	LoginLinkMessage:                       "<a href=\"%s\">Authorize Racó Bot</a>",
	ChoosePreferredLanguageMenuText:        "Please choose your preferred language:",
	LanguageUnavailableErrorMessage:        "<i>Language unavailable</i>",
	PreferredLanguageSetMessage:            "The preferred language has been set to English.",
}
