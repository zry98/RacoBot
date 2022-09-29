package bot

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strings"

	hr "github.com/coolspring8/go-lolhtml" // HTMLRewriter
	log "github.com/sirupsen/logrus"
	tb "gopkg.in/telebot.v3"

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
	user    db.User
	linkURL string
}

// Send sends a NoticeMessage
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
	racoBaseURL           string = "https://raco.fib.upc.edu"
	racoNoticeURLTemplate string = "https://raco.fib.upc.edu/avisos/veure.jsp?espai=%d&id=%d"
	datetimeLayout        string = "02/01/2006 15:04:05"
)

var (
	htmlCommentRegex = regexp.MustCompile(`<!--.*?-->`)
	// HTML tags currently supported in Telegram API
	supportedTagNames         = [...]string{"a", "b", "strong", "i", "em", "u", "ins", "s", "strike", "del", "code", "pre", "tg-spoiler"}
	topLevelListItemPrefix    = `  • `
	nestedListItemPrefix      = `    • `
	htmlEntityReplaceExcluder = strings.NewReplacer("&lt;", "&`lt;", "&gt;", "&`gt;", "&amp;", "&`amp;", "&quot;", "&`quot;")
	htmlEntityReplaceRestorer = strings.NewReplacer("&`lt;", "&lt;", "&`gt;", "&gt;", "&`amp;", "&amp;", "&`quot;", "&quot;")
	htmlRewriterHandlers      = hr.Handlers{
		ElementContentHandler: []hr.ElementContentHandler{
			{
				// add newline before exam title
				Selector: `div[class="extraInfo"]`,
				ElementHandler: func(e *hr.Element) hr.RewriterDirective {
					if err := e.InsertBeforeStartTagAsText("\n"); err != nil {
						log.Error(err)
						return hr.Stop
					}
					return hr.Continue
				},
			},
			{
				// add newline after exam time
				Selector: `span[id="horaExamen"]`,
				ElementHandler: func(e *hr.Element) hr.RewriterDirective {
					if err := e.InsertAfterEndTagAsText("\n"); err != nil {
						log.Error(err)
						return hr.Stop
					}
					return hr.Continue
				},
			},
			{
				// italicize exam info subtitle
				Selector: `span[class="label"]`,
				ElementHandler: func(e *hr.Element) hr.RewriterDirective {
					if err := e.RemoveAttribute("class"); err != nil {
						log.Error(err)
						return hr.Stop
					}
					if err := e.SetTagName("i"); err != nil {
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
				// fix underline
				Selector: `span[style="text-decoration:underline"]`,
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
				// fix link with path-only URL
				Selector: `a[href^="/"]`,
				ElementHandler: func(e *hr.Element) hr.RewriterDirective {
					href, err := e.AttributeValue("href")
					if err != nil {
						log.Error(err)
						return hr.Stop
					}
					if err = e.SetAttribute("href", racoBaseURL+href); err != nil {
						log.Error(err)
						return hr.Stop
					}
					return hr.Continue
				},
			},
			{
				// Telegram doesn't support `<br>` but `\n`
				Selector: `br`,
				ElementHandler: func(e *hr.Element) hr.RewriterDirective {
					if err := e.ReplaceAsText("\n"); err != nil {
						log.Error(err)
						return hr.Stop
					}
					return hr.Continue
				},
			},
			// Telegram doesn't support lists (`<ul>`, `<ol>` and `<li>`),
			// so we add a bullet point at the beginning of each item (`<li>`) as an indicator
			// the following three handlers are for different types of list items
			// TODO: add numbering to items in ordered lists
			{
				// item in nested list, prepend 4 spaces and a bullet point (`    • `)
				Selector: `li > ul > li`,
				ElementHandler: func(e *hr.Element) hr.RewriterDirective {
					if err := e.InsertBeforeStartTagAsText(nestedListItemPrefix); err != nil {
						log.Error(err)
						return hr.Stop
					}
					// add newline after each item
					if err := e.InsertAfterEndTagAsText("\n"); err != nil {
						log.Error(err)
						return hr.Stop
					}
					return hr.Continue
				},
			},
			{
				// item in nested list, prepend 4 spaces and a bullet point (`    • `)
				Selector: `li > ol > li`,
				ElementHandler: func(e *hr.Element) hr.RewriterDirective {
					if err := e.InsertBeforeStartTagAsText(nestedListItemPrefix); err != nil {
						log.Error(err)
						return hr.Stop
					}
					// add newline after each item
					if err := e.InsertAfterEndTagAsText("\n"); err != nil {
						log.Error(err)
						return hr.Stop
					}
					return hr.Continue
				},
			},
			{
				// item in top-level list, prepend 2 spaces and a bullet point (`  • `)
				Selector: `li`,
				ElementHandler: func(e *hr.Element) hr.RewriterDirective {
					if err := e.InsertBeforeStartTagAsText(topLevelListItemPrefix); err != nil {
						log.Error(err)
						return hr.Stop
					}
					// add newline after each item
					if err := e.InsertAfterEndTagAsText("\n"); err != nil {
						log.Error(err)
						return hr.Stop
					}
					return hr.Continue
				},
			},
			{
				// strip all unsupported tags
				Selector: `*`,
				ElementHandler: func(e *hr.Element) hr.RewriterDirective {
					tagName := e.TagName()
					for _, supportedTagName := range supportedTagNames {
						if tagName == supportedTagName {
							return func() hr.RewriterDirective { // strip all attributes the tag has
								it := e.AttributeIterator()
								defer it.Free()
								for {
									nextAttrib := it.Next()
									if nextAttrib == nil || (nextAttrib.Name() == "href" && e.TagName() == "a") {
										break
									}
									if err := e.RemoveAttribute(nextAttrib.Name()); err != nil {
										log.Error(err)
										return hr.Stop
									}
								}
								return hr.Continue
							}()
						}
					}
					e.RemoveAndKeepContent()
					return hr.Continue
				},
			},
		},
	}
)

// String formats a NoticeMessage to a proper string ready to be sent by bot
func (m *NoticeMessage) String() string {
	locale := locales.Get(m.user.LanguageCode)
	var sb strings.Builder

	// message header (subject, title, publish time, original link)
	header := fmt.Sprintf("[#%s] <b>%s</b>\n\n<i>%s</i>  %s",
		strings.ReplaceAll(strings.TrimPrefix(m.SubjectCode, "#"), "-", "_"), // telegram tags can't contain dashes
		m.Title,
		m.PublishedAt.Format(datetimeLayout),
		fmt.Sprintf("<a href=\"%s\">%s</a>", m.linkURL, locale.NoticeMessageOriginalLinkText))
	sb.WriteString(header)

	// format body text
	if m.Text != "" {
		text, err := hr.RewriteString(m.Text, &htmlRewriterHandlers)
		if err != nil {
			log.Errorf("error rewriting notice message text HTML: %v", err)
			return fmt.Sprintf("%s\n\n%s", header, locale.InternalErrorMessage)
		}

		// unescape HTML entities except `&lt;`, `&gt;`, `&amp;` and `&quot;`
		// FIXME: too janky
		text = htmlEntityReplaceExcluder.Replace(text)
		text = html.UnescapeString(text) // unescape other HTML entities
		text = htmlEntityReplaceRestorer.Replace(text)

		text = htmlCommentRegex.ReplaceAllString(text, "") // remove HTML comments
		text = strings.Trim(text, "\n\r")                  // remove trailing newlines
		sb.WriteString("\n\n")
		sb.WriteString(text)
	}

	// append attachments if there are any
	if len(m.Attachments) != 0 {
		noun := locale.NoticeMessageAttachmentNounSingular
		if len(m.Attachments) > 1 {
			noun = locale.NoticeMessageAttachmentNounPlural
			// sort attachments by filename
			sort.Slice(m.Attachments, func(i, j int) bool {
				return m.Attachments[i].Name < m.Attachments[j].Name
			})
		}

		var sbAttachments strings.Builder
		for _, attachment := range m.Attachments {
			fileSize := strings.ReplaceAll(byteCountIEC(attachment.Size), ".", string(locale.DecimalSeparator))
			fmt.Fprintf(&sbAttachments, "<a href=\"%s\">%s</a>  (%s)\n", attachment.RedirectURL, attachment.Name, fileSize)
		}
		fmt.Fprintf(&sb, "\n\n%s\n%s",
			fmt.Sprintf(locale.NoticeMessageAttachmentListHeader, len(m.Attachments), noun),
			strings.TrimSuffix(sbAttachments.String(), "\n"))
	}

	// send racó notice URL instead if message length exceeds the limit
	if sb.Len() > messageMaxLength {
		return fmt.Sprintf("%s\n\n%s",
			header,
			fmt.Sprintf(locale.NoticeMessageTooLongErrorMessage, m.linkURL))
	}
	return sb.String()
}

// byteCountIEC returns the human-readable file size of the given bytes count
func byteCountIEC(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// ErrorMessage represents a message containing error info
type ErrorMessage struct {
	Text string
}

// Send sends an ErrorMessage
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

// SilentMessage represents a message that should be sent with notification disabled
type SilentMessage struct {
	Text string
}

// Send sends a SilentMessage
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

// AnnouncementMessage represents an announcement message to be sent to all users
type AnnouncementMessage struct {
	Text string
}

// Send sends an AnnouncementMessage
func (m *AnnouncementMessage) Send(b *tb.Bot, to tb.Recipient, opt *tb.SendOptions) (*tb.Message, error) {
	params := map[string]string{
		"chat_id":    to.Recipient(),
		"text":       m.Text,
		"parse_mode": tb.ModeHTML,
	}

	data, err := b.Raw("sendMessage", params)
	if err != nil {
		return nil, err
	}

	msg, err := extractMessage(data)
	if err != nil {
		return nil, err
	}

	// pin it
	err = b.Pin(msg)
	return msg, err
}

// extractMessage extracts common Message result from given data.
// Should be called after extractOk or b.Raw() to handle possible errors.
// copied from telebot for implementing Sendable interfaces
// (Source: https://github.com/tucnak/telebot/blob/dd790ca6c1a5b187922415325a2cc2c66e033214/util.go#L110)
func extractMessage(data []byte) (*tb.Message, error) {
	var resp struct {
		Result *tb.Message
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		var resp struct {
			Result bool
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("telebot: %w", err)
		}
		if resp.Result {
			return nil, fmt.Errorf("telebot: result is True")
		}
		return nil, fmt.Errorf("telebot: %w", err)
	}
	return resp.Result, nil
}
