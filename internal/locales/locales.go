package locales

type Locale struct {
	StartMessage                           string
	GreetingMessage                        string
	HelpMessage                            string
	AlreadyLoggedInMessage                 string
	LogoutSucceededMessage                 string
	NoticeMessageAttachmentNounSingular    string
	NoticeMessageAttachmentNounPlural      string
	NoticeMessageAttachmentIndicator       string
	NoticeMessageTooLongErrorMessage       string
	NoticeUnavailableErrorMessage          string
	NoNoticesAvailableErrorMessage         string
	InternalErrorMessage                   string
	FIBAPIAuthorizationExpiredErrorMessage string
	LoginLinkMessage                       string
}

var default_locale *Locale

func init() {
	default_locale = &en
}

func Get(languageCode string) *Locale {
	switch languageCode {
	case "en":
		return &en
	case "es":
		return &es
	case "ca":
		return &ca
	default:
		return default_locale
	}
}
