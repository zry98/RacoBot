package fibapi

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// UserInfo represents a user's information API response
// Endpoint: /jo.json
type UserInfo struct {
	Username  string `json:"username"`
	FirstName string `json:"nom"`
	LastNames string `json:"cognoms"`
}

// NoticesResponse represents a user's notices API response
// Endpoint: /jo/avisos.json
type NoticesResponse struct {
	Count   int      `json:"count"`
	Results []Notice `json:"results"`
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
	PublishedAt TimeDate     `json:"__published_at,omitempty"`
}

// UnmarshalJSON adds .PublishedAt to the Notice when it's unmarshalled from JSON
func (n *Notice) UnmarshalJSON(b []byte) error {
	type Alias Notice
	tmp := (*Alias)(n)
	if err := json.Unmarshal(b, tmp); err != nil {
		return err
	}

	n.PublishedAt = n.ModifiedAt
	if n.ModifiedAt.Unix() < n.CreatedAt.Unix() {
		n.PublishedAt = n.CreatedAt
	}
	return nil
}

// Attachment represents a single attachment in a Notice's attachments
type Attachment struct {
	MimeTypes   string   `json:"tipus_mime"`
	Name        string   `json:"nom"`
	URL         string   `json:"url"`
	ModifiedAt  TimeDate `json:"data_modificacio"`
	Size        int64    `json:"mida"`
	RedirectURL string   `json:"__redirect_url,omitempty"`
}

// UnmarshalJSON adds .RedirectURL (the attachment's FIB API login redirect URL) to the Attachment when it's unmarshalled from JSON
// it's useful since FIB API cookies on the user's browser will expire, accessing an attachment's original URL after that will get an `Unauthorized` response
func (a *Attachment) UnmarshalJSON(b []byte) error {
	type Alias Attachment
	tmp := (*Alias)(a)
	if err := json.Unmarshal(b, tmp); err != nil {
		return err
	}

	a.RedirectURL = LoginRedirectBaseURL + url.QueryEscape(a.URL)
	return nil
}

// ScheduleResponse represents a user's schedule API response
// Endpoint: /jo/classes.json
type ScheduleResponse struct {
	Count   int     `json:"count"`
	Results []Class `json:"results"`
}

// Class represents a single class in a ScheduleResponse API response
type Class struct {
	SubjectCode string `json:"codi_assig"`
	Group       string `json:"grup"`
	DayOfWeek   int    `json:"dia_setmana"`
	StartTime   string `json:"inici"`
	Duration    int    `json:"durada"`
	Types       string `json:"tipus"`
	Classrooms  string `json:"aules"`
}

// SubjectsResponse represents a user's subjects API response
// Endpoint: /jo/assignatures.json
type SubjectsResponse struct {
	Count   int       `json:"count"`
	Results []Subject `json:"results"`
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

// TimeDate represents the time&date data in API response JSONs
type TimeDate struct {
	time.Time
}

const timeDateLayout = "2006-01-02T15:04:05"

var (
	tzMadrid *time.Location
	nilTime  = (time.Time{}).Unix()
)

// init initializes the Madrid timezone used for parsing time&date in FIB API response JSONs
func init() {
	var err error
	tzMadrid, err = time.LoadLocation("Europe/Madrid")
	if err != nil {
		panic(err)
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface for TimeDate type
// it un-marshals the `2006-01-02T15:04:05`-format time&date strings to TimeDate type
func (t *TimeDate) UnmarshalJSON(b []byte) (err error) {
	t.Time, err = time.ParseInLocation(timeDateLayout, strings.Trim(string(b), `"`), tzMadrid)
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

// PublicSubject represents a PublicSubject's API response, or a single subject in a PublicSubjectsResponse
// Endpoint: /assignatures/{acronym}.json
type PublicSubject struct {
	ID           string   `json:"id"`
	Name         string   `json:"nom"`
	Acronym      string   `json:"sigles"`
	UPCCode      int64    `json:"codi_upc"`
	URL          string   `json:"url"`
	GuideURL     string   `json:"guia"`
	Plans        []string `json:"plans"`
	Obligatories []struct {
		ObligatoryCode string `json:"codi_oblig"`
		SpecialityCode string `json:"codi_especialitat"`
		SpecialityName string `json:"nom_especialitat"`
		Plan           string `json:"pla"`
	} `json:"obligatorietats"`
	Languages    map[string][]string `json:"lang"`
	Semesters    []string            `json:"quadrimestres"`
	Semester     string              `json:"semestre"`
	Credits      float32             `json:"credits"`
	Availability string              `json:"vigent"`
}
