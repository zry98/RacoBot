package locales

import tb "gopkg.in/telebot.v3"

// Locale represents a locale (group of translations)
type Locale struct {
	DecimalSeparator                    rune
	StartMessage                        string
	LoginLinkMessage                    string
	GreetingMessage                     string
	HelpMessage                         string
	AlreadyLoggedInMessage              string
	LogoutSucceededMessage              string
	LogoutFailedMessage                 string
	NoticeMessageOriginalLinkText       string
	NoticeMessageAttachmentNounSingular string
	NoticeMessageAttachmentNounPlural   string
	NoticeMessageAttachmentListHeader   string
	NoticeMessageTooLongErrorMessage    string
	NoticeUnavailableErrorMessage       string
	NoAvailableNoticesErrorMessage      string
	InternalErrorMessage                string
	FIBAPIAuthorizationExpiredMessage   string
	SelectPreferredLanguageMenuText     string
	LanguageUnavailableErrorMessage     string
	PreferredLanguageSetMessage         string
	//Authorized                          string
	//AuthorizedResponseMessage           string
	CommandsMenu []tb.Command
}

var LanguageCodes = [...]string{"ca", "es", "en"}

// Get returns a Locale by the given language code
func Get(languageCode string) *Locale {
	switch languageCode {
	case "es":
		return &es
	case "ca":
		return &ca
	case "en":
		return &en
	default:
		return &es // default to castellano
	}
}
