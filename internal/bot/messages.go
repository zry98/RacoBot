package bot

import (
	"fmt"
	"strings"

	hr "github.com/coolspring8/go-lolhtml" // HTMLRewriter
	log "github.com/sirupsen/logrus"

	"RacoBot/internal/db"
	"RacoBot/internal/locales"
	"RacoBot/pkg/fibapi"
)

// TODO: implement Messages with the `telebot.Sendable` interface?

// LoginLinkMessage represents a link message of a login (FIB API authorization) session
type LoginLinkMessage struct {
	db.LoginSession
}

// String formats a LoginLinkMessage to a proper string with a generated Authorization URL ready to be sent by bot
func (m LoginLinkMessage) String() string {
	authorizationURL := fibapi.NewAuthorizationURL(m.State)
	return fmt.Sprintf(locales.Get(m.UserLanguageCode).LoginLinkMessage, authorizationURL)
}

// NoticeMessage represents a FIB API Notice message
type NoticeMessage struct {
	fibapi.Notice
	user db.User
}

const (
	messageMaxLength      int    = 4096
	RacoNoticeURLTemplate string = "https://raco.fib.upc.edu/avisos/veure.jsp?assig=GRAU-%s&id=%d" // TODO: use `espai` parameter (UPC subject code)
)

// these are the HTML tags Telegram supported
var supportedTagNames = [...]string{"a", "b", "strong", "i", "em", "u", "ins", "s", "strike", "del", "code", "pre"}

// String formats a NoticeMessage to a proper string ready to be sent by bot
func (n NoticeMessage) String() (result string) {
	locale := locales.Get(n.user.LanguageCode)

	if n.Text != "" {
		var err error
		result, err = hr.RewriteString(
			n.Text,
			&hr.Handlers{
				ElementContentHandler: []hr.ElementContentHandler{
					{
						Selector: "div[class='extraInfo']",
						// add newlines before exam info titles
						ElementHandler: func(e *hr.Element) hr.RewriterDirective {
							if err := e.InsertBeforeStartTagAsText("\n"); err != nil {
								log.Error(err)
								return hr.Stop
							}
							return hr.Continue
						},
					},
					{
						Selector: "span[id='horaExamen']",
						// add newlines after exam time data
						ElementHandler: func(e *hr.Element) hr.RewriterDirective {
							if err := e.InsertAfterEndTagAsText("\n"); err != nil {
								log.Error(err)
								return hr.Stop
							}
							return hr.Continue
						},
					},
					{
						Selector: "span[class='label']",
						// italicize info titles
						ElementHandler: func(e *hr.Element) hr.RewriterDirective {
							if err := e.SetTagName("i"); err != nil {
								log.Error(err)
								return hr.Stop
							}
							if err := e.RemoveAttribute("class"); err != nil {
								log.Error(err)
								return hr.Stop
							}
							if err := e.InsertBeforeStartTagAsHTML("- "); err != nil {
								log.Error(err)
								return hr.Stop
							}
							return hr.Continue
						},
					},
					{
						Selector: "span[style='text-decoration:underline']",
						// underlines
						ElementHandler: func(e *hr.Element) hr.RewriterDirective {
							if err := e.RemoveAttribute("style"); err != nil {
								log.Error(err)
								return hr.Stop
							}
							if err := e.SetTagName("u"); err != nil {
								log.Error(err)
								return hr.Stop
							}
							return hr.Continue
						},
					},
					{
						Selector: "br",
						// Telegram doesn't support <br> but \n
						ElementHandler: func(e *hr.Element) hr.RewriterDirective {
							err := e.ReplaceAsText("\n")
							if err != nil {
								log.Error(err)
								return hr.Stop
							}
							return hr.Continue
						},
					},
					{
						Selector: "li",
						// Telegram doesn't support <ul> & <li>, so add a `- ` at the beginning as an indicator
						ElementHandler: func(e *hr.Element) hr.RewriterDirective {
							err := e.InsertBeforeStartTagAsText("- ")
							if err != nil {
								log.Error(err)
								return hr.Stop
							}
							err = e.InsertAfterEndTagAsText("\n") // newline after each entry
							if err != nil {
								log.Error(err)
								return hr.Stop
							}
							return hr.Continue
						},
					},
					{
						Selector: "*",
						// strip all the other tags since Telegram doesn't support them
						ElementHandler: func(e *hr.Element) hr.RewriterDirective {
							tagName := e.TagName()
							for _, supportedTagName := range supportedTagNames {
								if tagName == supportedTagName {
									return hr.Continue
								}
							}
							e.RemoveAndKeepContent()
							return hr.Continue
						},
					},
				},
			},
		)
		if err != nil {
			log.Fatal(err)
			return fmt.Sprintf("<i>Internal error</i>\nNotice ID: %d", n.ID)
		}
		result = "\n\n" + result
	}

	// TODO: use template
	result = fmt.Sprintf("[%s] <b>%s</b>%s",
		n.SubjectCode,
		n.Title,
		result)

	if len(n.Attachments) != 0 {
		var sb strings.Builder
		for _, attachment := range n.Attachments {
			fileSize := byteCountIEC(attachment.Size)
			fmt.Fprintf(&sb, "<a href=\"%s\">%s</a>  (%s)\n", attachment.RedirectURL(), attachment.Name, fileSize)
		}

		noun := locale.NoticeMessageAttachmentNounPlural
		if len(n.Attachments) > 1 {
			noun = locale.NoticeMessageAttachmentNounPlural
		}
		result = fmt.Sprintf(locale.NoticeMessageAttachmentIndicator, result, len(n.Attachments), noun, sb.String())
	}

	if len(result) > messageMaxLength { // send Racó notice URL instead if message length exceeds the limit of 4096 characters
		result = fmt.Sprintf(locale.NoticeMessageTooLongErrorMessage,
			n.SubjectCode,
			n.Title,
			fmt.Sprintf(RacoNoticeURLTemplate, n.SubjectCode, n.ID))
	}
	return result
}

// byteCountIEC returns the human-readable file size of the given bytes count
func byteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
