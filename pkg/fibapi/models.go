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

// Notice represents a single notice in a NoticesResponse API response
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

// NoticesResponse represents a user's notices API response
// Endpoint: /jo/avisos
type NoticesResponse struct {
	Count   int      `json:"count"`
	Results []Notice `json:"results"`
}

// Schedule represents a single schedule in a SchedulesResponse API response
type Schedule struct {
	SubjectCode string `json:"codi_assig"`
	Group       string `json:"grup"`
	DayOfWeek   int    `json:"dia_setmana"`
	StartTime   string `json:"inici"`
	Duration    int    `json:"durada"`
	Types       string `json:"tipus"`
	Classrooms  string `json:"aules"`
}

// SchedulesResponse represents a user's schedules API response
// Endpoint: /jo/classes
type SchedulesResponse struct {
	Count   int        `json:"count"`
	Results []Schedule `json:"results"`
}

// Subject represents a single subject in a SubjectsResponse API response
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

// SubjectsResponse represents a user's subjects API response
// Endpoint: /jo/assignatures
type SubjectsResponse struct {
	Count   int       `json:"count"`
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

// RedirectURL returns an attachment's FIB API login redirect URL
// it's useful since FIB API cookies on the user's browser will expire, accessing an attachment's original URL after that will get an `Unauthorized` response
func (a Attachment) RedirectURL() string {
	return LoginRedirectBaseURL + url.QueryEscape(a.URL)
}

// TimeDate represents the time&date data in API response JSONs
type TimeDate struct {
	time.Time
}

const timeDateLayout = "2006-01-02T15:04:05"

var nilTime = (time.Time{}).Unix()

// UnmarshalJSON implements the json.Unmarshaler interface for TimeDate type
// it un-marshals the `2006-01-02T15:04:05`-format time&date strings in the API response JSONs to TimeDate type
func (t *TimeDate) UnmarshalJSON(b []byte) error {
	tzMadrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		return err
	}

	t.Time, err = time.ParseInLocation(timeDateLayout, strings.Trim(string(b), "\""), tzMadrid)
	return err
}

// MarshalJSON implements the json.Marshaler interface for TimeDate type
// it marshals the values of fields with TimeDate type to UNIX timestamp format
func (t TimeDate) MarshalJSON() ([]byte, error) {
	if t.Time.Unix() == nilTime {
		return []byte("nil"), nil
	}
	return []byte(strconv.FormatInt(t.Time.Unix(), 10)), nil
}
