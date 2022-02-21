package locales

import tb "gopkg.in/telebot.v3"

// Locale represents a locale (group of translations)
type Locale struct {
	StartMessage                           string
	AuthorizedResponseBody                 string
	GreetingMessage                        string
	HelpMessage                            string
	AlreadyLoggedInMessage                 string
	LogoutSucceededMessage                 string
	NoticeMessageAttachmentNounSingular    string
	NoticeMessageAttachmentNounPlural      string
	NoticeMessageAttachmentIndicator       string
	NoticeMessageTooLongErrorMessage       string
	NoticeUnavailableErrorMessage          string
	NoAvailableNoticesErrorMessage         string
	InternalErrorMessage                   string
	FIBAPIAuthorizationExpiredErrorMessage string
	LoginLinkMessage                       string
	ChoosePreferredLanguageMenuText        string
	LanguageUnavailableErrorMessage        string
	PreferredLanguageSetMessage            string
	DecimalSeparator                       rune
	CommandsMenu                           []tb.Command
}

var defaultLocale *Locale

func init() {
	defaultLocale = &es
}

// Get returns a Locale by the given language code
func Get(languageCode string) *Locale {
	switch languageCode {
	case "ca":
		return &ca
	case "es":
		return &es
	case "en":
		return &en
	default:
		return defaultLocale
	}
}
