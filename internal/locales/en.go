package locales

import tb "gopkg.in/telebot.v3"

var en = Locale{
	StartMessage:                        "Please /login to authorize Racó Bot.",
	Authorized:                          "Authorized",
	AuthorizedResponseMessage:           "You can now close this browser tab and return to Telegram.",
	GreetingMessage:                     "Hello, %s!",
	HelpMessage:                         "You can use:\n/test to preview the latest one notice.\n/logout to stop receiving messages and revoke the authorization on server.",
	AlreadyLoggedInMessage:              "You are already logged-in, check /whoami; or /logout to revoke the authorization.",
	LogoutSucceededMessage:              "You have successfully logged out! And the FIB API token has been revoked on server.",
	LogoutFailedMessage:                 `An error has occurred while logging you out. Although the bot has already deleted you from the database, you can revoke the token manually on <a href="https://api.fib.upc.edu/v2/o/authorized_tokens/">the FIB API</a> if you want.`,
	NoticeMessageOriginalLinkText:       "Link",
	NoticeMessageAttachmentNounSingular: "attachment",
	NoticeMessageAttachmentNounPlural:   "attachments",
	NoticeMessageAttachmentListHeader:   "📎 <i>With %d %s:</i>",
	NoticeMessageTooLongErrorMessage:    "🤖 <i>Sorry, but this message is too long to be sent by Telegram, please view it through <a href=\"%s\">this link</a>.<i>",
	NoticeUnavailableErrorMessage:       "<i>Notice unavailable</i>",
	NoAvailableNoticesErrorMessage:      "<i>No available notices</i>",
	InternalErrorMessage:                "🤖 <i>An internal error has occurred</i>",
	FIBAPIAuthorizationExpiredMessage:   "Authorization has expired, please /login again.",
	LoginLinkMessage:                    "<a href=\"%s\">Authorize Racó Bot</a>",
	SelectPreferredLanguageMenuText:     "Please select your preferred language:",
	LanguageUnavailableErrorMessage:     "<i>Language unavailable</i>",
	PreferredLanguageSetMessage:         "Your preferred language has been set to English.",
	DecimalSeparator:                    '.',
	CommandsMenu: []tb.Command{
		{Text: "login", Description: "Authorize bot on FIB API"},
		{Text: "lang", Description: "Select preferred language"},
		{Text: "whoami", Description: "Show personal information"},
		{Text: "test", Description: "Show the latest one notice"},
		{Text: "logout", Description: "De-authorize bot"},
	},
}
