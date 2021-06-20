package fibapi

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

// UserInfo represents a user's information API response
// Endpoint: /jo
type UserInfo struct {
	Username  string `json:"username"`
	FirstName string `json:"nom"`
	LastNames string `json:"cognoms"`
}

// Notice represents a single notice in a Notices API response
type Notice struct {
	ID          int64        `json:"id"`
	Title       string       `json:"titol"`
	SubjectCode string       `json:"codi_assig"`
	Text        string       `json:"text"`
	CreatedAt   TimeDate     `json:"data_insercio"`
	ModifiedAt  TimeDate     `json:"data_modificacio"`
	ExpiresAt   TimeDate     `json:"data_caducitat"`
	Attachments []Attachment `json:"adjunts"`
}

// Notices represents a user's notices API response
// Endpoint: /jo/avisos
type Notices struct {
	Count   int      `json:"count"`
	Results []Notice `json:"results"`
}

// Schedule represents a single schedule in a Schedules API response
type Schedule struct {
	SubjectCode string `json:"codi_assig"`
	Group       string `json:"grup"`
	DayOfWeek   int    `json:"dia_setmana"`
	StartTime   string `json:"inici"`
	Duration    int    `json:"durada"`
	Types       string `json:"tipus"`
	Classrooms  string `json:"aules"`
}

// Schedules represents a user's schedules API response
// Endpoint: /jo/classes
type Schedules struct {
	Count   int        `json:"count"`
	Results []Schedule `json:"results"`
}

// Subject represents a single subject in a Subjects API response
type Subject struct {
	ID       string  `json:"id"`
	URL      string  `json:"url"`
	GuideURL string  `json:"guia"`
	Group    string  `json:"grup"`
	Acronym  string  `json:"sigles"`
	UPCCode  int     `json:"codi_upc"`
	Semester string  `json:"semestre"`
	Credits  float32 `json:"credits"`
	Name     string  `json:"nom"`
}

// Subjects represents a user's subjects API response
// Endpoint: /jo/assignatures
type Subjects struct {
	Count   int       `json:"id"`
	Results []Subject `json:"results"`
}

// Attachment represents a single attachment in a Notice's attachments
type Attachment struct {
	MimeTypes  string   `json:"tipus_mime"`
	Name       string   `json:"nom"`
	URL        string   `json:"url"`
	ModifiedAt TimeDate `json:"data_modificacio"`
	Size       int64    `json:"mida"`
}

const LoginRedirectBaseURL = "https://api.fib.upc.edu/v2/accounts/login/?next="

// RedirectURL returns an attachment's FIB API login redirect URL
// it's useful since FIB API cookies on the user's browser will expire, accessing an attachment's original URL after that will get an `Unauthorized` response
func (a Attachment) RedirectURL() string {
	// FIXME: waiting for the suffix bug in FIB API to be fixed
	// the bug: attachments' URLs will have the `.json` suffixes if requested endpoint was `/jo/avisos.json`
	return LoginRedirectBaseURL + url.QueryEscape(strings.TrimSuffix(a.URL, `.json`))
}

// TimeDate represents the time&date data in API response JSONs
type TimeDate time.Time

// UnmarshalJSON implements the json.Unmarshaler interface for TimeDate type
// it un-marshals the `2006-01-02T15:04:05`-format time&date strings in the API response JSONs to TimeDate type
func (j *TimeDate) UnmarshalJSON(b []byte) (err error) {
	tzMadrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		return
	}

	t, err := time.ParseInLocation("2006-01-02T15:04:05", strings.Trim(string(b), "\""), tzMadrid)
	if err != nil {
		return
	}

	*j = TimeDate(t)
	return
}

// MarshalJSON implements the json.Marshaler interface for TimeDate type
// it marshals the values of fields with TimeDate type to UNIX timestamp format
func (j TimeDate) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(time.Time(j).Unix(), 10)), nil
}
