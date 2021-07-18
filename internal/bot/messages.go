package bot

import (
	"encoding/json"
	"fmt"
	"strings"

	hr "github.com/coolspring8/go-lolhtml" // HTMLRewriter
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	tb "gopkg.in/tucnak/telebot.v3"

	"RacoBot/internal/db"
	"RacoBot/internal/locales"
	"RacoBot/pkg/fibapi"
)

// LoginLinkMessage represents a link message of a login (FIB API authorization) session
type LoginLinkMessage struct {
	db.LoginSession
}

// Send formats a LoginLinkMessage to a proper string with a generated Authorization URL and sends it
func (m *LoginLinkMessage) Send(b *tb.Bot, to tb.Recipient, opt *tb.SendOptions) (*tb.Message, error) {
	authorizationURL := fibapi.NewAuthorizationURL(m.State)
	text := fmt.Sprintf(locales.Get(m.UserLanguageCode).LoginLinkMessage, authorizationURL)

	params := map[string]string{
		"chat_id":    to.Recipient(),
		"text":       text,
		"parse_mode": tb.ModeHTML,
	}

	data, err := b.Raw("sendMessage", params)
	if err != nil {
		return nil, err
	}

	return extractMessage(data)
}

// NoticeMessage represents a FIB API Notice message
type NoticeMessage struct {
	fibapi.Notice
	user db.User
}

func (m *NoticeMessage) Send(b *tb.Bot, to tb.Recipient, opt *tb.SendOptions) (*tb.Message, error) {
	params := map[string]string{
		"chat_id":                  to.Recipient(),
		"text":                     m.String(),
		"parse_mode":               tb.ModeHTML,
		"disable_web_page_preview": "true",
	}

	data, err := b.Raw("sendMessage", params)
	if err != nil {
		return nil, err
	}

	return extractMessage(data)
}

const (
	messageMaxLength      int    = 4096
	RacoNoticeURLTemplate string = "https://raco.fib.upc.edu/avisos/veure.jsp?assig=GRAU-%s&id=%d" // TODO: use `espai` parameter (UPC subject code)
)

// these are the HTML tags Telegram supported
var supportedTagNames = [...]string{"a", "b", "strong", "i", "em", "u", "ins", "s", "strike", "del", "code", "pre"}

// String formats a NoticeMessage to a proper string ready to be sent by bot
func (m *NoticeMessage) String() (result string) {
	locale := locales.Get(m.user.LanguageCode)

	if m.Text != "" {
		var err error
		result, err = hr.RewriteString(
			m.Text,
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
						// Telegram doesn't support <br> but \m
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
			return fmt.Sprintf("<i>Internal error</i>\nNotice ID: %d", m.ID)
		}
		result = "\n\n" + result
	}

	// TODO: use template
	result = fmt.Sprintf("[%s] <b>%s</b>%s",
		m.SubjectCode,
		m.Title,
		result)

	if len(m.Attachments) != 0 {
		var sb strings.Builder
		for _, attachment := range m.Attachments {
			fileSize := byteCountIEC(attachment.Size)
			fmt.Fprintf(&sb, "<a href=\"%s\">%s</a>  (%s)\n", attachment.RedirectURL(), attachment.Name, fileSize)
		}

		noun := locale.NoticeMessageAttachmentNounPlural
		if len(m.Attachments) > 1 {
			noun = locale.NoticeMessageAttachmentNounPlural
		}
		result = fmt.Sprintf(locale.NoticeMessageAttachmentIndicator, result, len(m.Attachments), noun, sb.String())
	}

	if len(result) > messageMaxLength { // send Racó notice URL instead if message length exceeds the limit of 4096 characters
		result = fmt.Sprintf(locale.NoticeMessageTooLongErrorMessage,
			m.SubjectCode,
			m.Title,
			fmt.Sprintf(RacoNoticeURLTemplate, m.SubjectCode, m.ID))
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

type ErrorMessage struct {
	Text string
}

func (m *ErrorMessage) Send(b *tb.Bot, to tb.Recipient, opt *tb.SendOptions) (*tb.Message, error) {
	params := map[string]string{
		"chat_id":                  to.Recipient(),
		"text":                     m.Text,
		"parse_mode":               tb.ModeHTML,
		"disable_web_page_preview": "true",
	}

	data, err := b.Raw("sendMessage", params)
	if err != nil {
		return nil, err
	}

	return extractMessage(data)
}

type SilentMessage struct {
	Text string
}

func (m *SilentMessage) Send(b *tb.Bot, to tb.Recipient, opt *tb.SendOptions) (*tb.Message, error) {
	params := map[string]string{
		"chat_id":              to.Recipient(),
		"text":                 m.Text,
		"disable_notification": "true",
	}

	data, err := b.Raw("sendMessage", params)
	if err != nil {
		return nil, err
	}

	return extractMessage(data)
}

var (
	ErrTrueResult = errors.New("telebot: result is True")
)

// extractMessage extracts common Message result from given data.
// Should be called after extractOk or b.Raw() to handle possible errors.
func extractMessage(data []byte) (*tb.Message, error) {
	var resp struct {
		Result *tb.Message
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		var resp struct {
			Result bool
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, errors.Wrap(err, "telebot")
		}
		if resp.Result {
			return nil, ErrTrueResult
		}
		return nil, errors.Wrap(err, "telebot")
	}
	return resp.Result, nil
}
