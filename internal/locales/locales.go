package locales

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
}

var defaultLocale *Locale

func init() {
	defaultLocale = &en
}

// Get returns a Locale by the given language code
func Get(languageCode string) *Locale {
	switch languageCode {
	case "en":
		return &en
	case "es":
		return &es
	case "ca":
		return &ca
	default:
		return defaultLocale
	}
}
