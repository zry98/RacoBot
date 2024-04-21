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
	Count   uint32   `json:"count"`
	Results []Notice `json:"results"`
}

// Notice represents a single notice in a NoticesResponse API response
type Notice struct {
	ID          int32        `json:"id"` // FIXME: unsigned?
	CreatedAt   Time         `json:"data_insercio"`
	ModifiedAt  Time         `json:"data_modificacio"`
	ExpiresAt   Time         `json:"data_caducitat"`
	PublishedAt Time         `json:"__published_at,omitempty"`
	SubjectCode string       `json:"codi_assig"`
	Title       string       `json:"titol"`
	Text        string       `json:"text"`
	Attachments []Attachment `json:"adjunts"`
}

// UnmarshalJSON adds .PublishedAt to the Notice when it's unmarshalled from JSON
func (n *Notice) UnmarshalJSON(b []byte) error {
	type Alias Notice
	aux := (*Alias)(n)
	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}

	if n.CreatedAt.Unix() > n.ModifiedAt.Unix() {
		n.PublishedAt = n.CreatedAt
	} else {
		n.PublishedAt = n.ModifiedAt
	}
	return nil
}

// Attachment represents a single attachment in a Notice's attachments
type Attachment struct {
	Size        uint64 `json:"mida"`
	ModifiedAt  Time   `json:"data_modificacio"`
	Name        string `json:"nom"`
	URL         string `json:"url"`
	MimeTypes   string `json:"tipus_mime"`
	RedirectURL string `json:"__redirect_url,omitempty"`
}

// UnmarshalJSON adds .RedirectURL (the attachment's FIB API login redirect URL) to the Attachment when it's unmarshalled from JSON
// it's useful since FIB API cookies on the user's browser will expire, accessing an attachment's original URL after that will get an `Unauthorized` response
func (a *Attachment) UnmarshalJSON(b []byte) error {
	type Alias Attachment
	aux := (*Alias)(a)
	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}

	a.RedirectURL = loginRedirectBaseURL + url.QueryEscape(a.URL)
	return nil
}

// ScheduleResponse represents a user's schedule API response
// Endpoint: /jo/classes.json
type ScheduleResponse struct {
	Count   uint32  `json:"count"`
	Results []Class `json:"results"`
}

// Class represents a single class in a ScheduleResponse API response
type Class struct {
	DayOfWeek   uint8  `json:"dia_setmana"`
	Duration    uint8  `json:"durada"`
	SubjectCode string `json:"codi_assig"`
	Group       string `json:"grup"`
	StartTime   string `json:"inici"`
	Types       string `json:"tipus"`
	Classrooms  string `json:"aules"`
}

// SubjectsResponse represents a user's subjects API response
// Endpoint: /jo/assignatures.json
type SubjectsResponse struct {
	Count   uint32    `json:"count"`
	Results []Subject `json:"results"`
}

// Subject represents a single subject in a SubjectsResponse API response
type Subject struct {
	UPCCode  uint32  `json:"codi_upc,string"` // FIXME: signed?
	Credits  float32 `json:"credits"`
	ID       string  `json:"id"`
	Name     string  `json:"nom"`
	Acronym  string  `json:"sigles"`
	URL      string  `json:"url"`
	GuideURL string  `json:"guia"`
	Group    string  `json:"grup"`
	Semester string  `json:"semestre"`
}

// Time represents the time&date data in API response JSONs
type Time struct {
	time.Time
}

const timeDateLayout = "2006-01-02T15:04:05"

var tzMadrid *time.Location

// init initializes the Madrid timezone used for parsing time&date in FIB API response JSONs
func init() {
	var err error
	if tzMadrid, err = time.LoadLocation("Europe/Madrid"); err != nil {
		panic(err)
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface for Time type
// it un-marshals the `2006-01-02T15:04:05`-format time&date strings to Time type
func (t *Time) UnmarshalJSON(b []byte) (err error) {
	t.Time, err = time.ParseInLocation(timeDateLayout, strings.Trim(string(b), `"`), tzMadrid)
	return err
}

// MarshalJSON implements the json.Marshaler interface for Time type
// it marshals the values of fields with Time type to UNIX timestamp format
func (t Time) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(t.Time.Unix(), 10)), nil
}

// PublicSubject represents a PublicSubject's API response, or a single subject in a PublicSubjectsResponse
// Endpoint: /assignatures/{acronym}.json
type PublicSubject struct {
	UPCCode      uint32              `json:"codi_upc,string"`
	Credits      float32             `json:"credits"`
	ID           string              `json:"id"`
	Name         string              `json:"nom"`
	Acronym      string              `json:"sigles"`
	URL          string              `json:"url"`
	GuideURL     string              `json:"guia"`
	Semester     string              `json:"semestre"`
	Availability string              `json:"vigent"`
	Plans        []string            `json:"plans"`
	Semesters    []string            `json:"quadrimestres"`
	Languages    map[string][]string `json:"lang"`
	Obligatories []struct {
		ObligatoryCode string `json:"codi_oblig"`
		SpecialityCode string `json:"codi_especialitat"`
		SpecialityName string `json:"nom_especialitat"`
		Plan           string `json:"pla"`
	} `json:"obligatorietats"`
}

// PublicSubjectsResponse represents a public subjects API response
// Endpoint: /jo/assignatures.json
type PublicSubjectsResponse struct {
	Count       uint32          `json:"count"`
	Year        string          `json:"curs,omitempty"`
	NextURL     string          `json:"next,omitempty"`
	PreviousURL string          `json:"previous,omitempty"`
	Results     []PublicSubject `json:"results"`
}
