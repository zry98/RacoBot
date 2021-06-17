package bot

import (
	"fmt"
	"strings"

	"github.com/coolspring8/go-lolhtml"
	log "github.com/sirupsen/logrus"

	"RacoBot/internal/db"
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
	return fmt.Sprintf("<a href=\"%s\">Authorize RacóBot</a>", authorizationURL)
}

// NoticeMessage represents a FIB API Notice message
type NoticeMessage struct {
	fibapi.Notice
}

const (
	messageMaxLength      int    = 4096
	RacoNoticeURLTemplate string = "https://raco.fib.upc.edu/avisos/veure.jsp?assig=GRAU-%s&id=%d" // TODO: use `espai` parameter (UPC subject code)
)

// these are the HTML tags Telegram supported
var supportedTagNames = [...]string{"a", "b", "strong", "i", "em", "u", "ins", "s", "strike", "del", "code", "pre"}

// String formats a NoticeMessage to a proper string ready to be sent by bot
func (n NoticeMessage) String() (result string) {
	if n.Text != "" {
		var err error
		result, err = lolhtml.RewriteString(
			n.Text,
			&lolhtml.Handlers{
				ElementContentHandler: []lolhtml.ElementContentHandler{
					{
						Selector: "div[class='extraInfo']",
						// add newlines before exam info titles
						ElementHandler: func(e *lolhtml.Element) lolhtml.RewriterDirective {
							err := e.InsertBeforeStartTagAsText("\n")
							if err != nil {
								log.Error(err)
								return lolhtml.Stop
							}
							return lolhtml.Continue
						},
					},
					{
						Selector: "span[id='horaExamen']",
						// add newlines after exam time data
						ElementHandler: func(e *lolhtml.Element) lolhtml.RewriterDirective {
							err := e.InsertAfterEndTagAsText("\n")
							if err != nil {
								log.Error(err)
								return lolhtml.Stop
							}
							return lolhtml.Continue
						},
					},
					{
						Selector: "span[class='label']",
						// italicize info titles
						ElementHandler: func(e *lolhtml.Element) lolhtml.RewriterDirective {
							err := e.SetTagName("i")
							if err != nil {
								log.Error(err)
								return lolhtml.Stop
							}
							err = e.RemoveAttribute("class")
							if err != nil {
								log.Error(err)
								return lolhtml.Stop
							}
							err = e.InsertBeforeStartTagAsHTML("- ")
							if err != nil {
								log.Error(err)
								return lolhtml.Stop
							}
							return lolhtml.Continue
						},
					},
					{
						Selector: "br",
						// Telegram doesn't support <br> but \n
						ElementHandler: func(e *lolhtml.Element) lolhtml.RewriterDirective {
							err := e.ReplaceAsText("\n")
							if err != nil {
								log.Error(err)
								return lolhtml.Stop
							}
							return lolhtml.Continue
						},
					},
					{
						Selector: "li",
						// Telegram doesn't support <ul> & <li>, so add a `- ` at the beginning as an indicator
						ElementHandler: func(e *lolhtml.Element) lolhtml.RewriterDirective {
							err := e.InsertBeforeStartTagAsText("- ")
							if err != nil {
								log.Error(err)
								return lolhtml.Stop
							}
							err = e.InsertAfterEndTagAsText("\n") // newline after each entry
							if err != nil {
								log.Error(err)
								return lolhtml.Stop
							}
							return lolhtml.Continue
						},
					},
					{
						Selector: "*",
						// strip all the other tags since Telegram doesn't support them
						ElementHandler: func(e *lolhtml.Element) lolhtml.RewriterDirective {
							tagName := e.TagName()
							for _, supportedTagName := range supportedTagNames {
								if tagName == supportedTagName {
									return lolhtml.Continue
								}
							}
							e.RemoveAndKeepContent()
							return lolhtml.Continue
						},
					},
				},
			},
		)
		if err != nil {
			log.Fatal(err)
			return fmt.Sprintf("<i>Internal error</i>\nNotice ID: %d", n.ID)
		}
	}

	// TODO: use template
	result = fmt.Sprintf("[%s] <b>%s</b>\n\n%s",
		n.SubjectCode,
		n.Title,
		result)

	if len(n.Attachments) != 0 {
		var sb strings.Builder
		for _, attachment := range n.Attachments {
			fileSize := byteCountIEC(attachment.Size)
			fmt.Fprintf(&sb, "<a href=\"%s\">%s</a>  (%s)\n", attachment.RedirectURL(), attachment.Name, fileSize)
		}

		noun := "attachment"
		if len(n.Attachments) > 1 {
			noun += "s"
		}
		result = fmt.Sprintf("%s\n\n<i>- With %d %s:</i>\n%s", result, len(n.Attachments), noun, sb.String())
	}

	if len(result) > messageMaxLength { // send Racó notice URL instead if message length exceeds the limit of 4096 characters
		result = fmt.Sprintf("[%s] <b>%s</b>\n\nSorry, but this message is too long to be sent through Telegram, please view it through <a href=\"%s\">this link</a>.",
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
