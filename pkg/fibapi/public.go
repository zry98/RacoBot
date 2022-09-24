package fibapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

var (
	publicClient   *http.Client
	publicClientID string
)

// GetPublicSubjects gets all subjects from the public API
func GetPublicSubjects() (subjects []PublicSubject, err error) {
	var expectedTotal uint32 = 0

	URL := publicSubjectsURL
	for { // loop until all pages are fetched
		body, _, e := requestPublic(http.MethodGet, URL)
		if e != nil {
			err = fmt.Errorf("error getting PublicSubjects: %w", e)
			return
		}

		var resp PublicSubjectsResponse
		if err = json.Unmarshal(body, &resp); err != nil {
			err = fmt.Errorf("error parsing PublicSubjects: %w\n%s", err, string(body))
			return
		}

		if expectedTotal == 0 {
			expectedTotal = resp.Count
		}
		subjects = append(subjects, resp.Results...)

		if resp.NextURL == "" { // all fetched
			break
		} else { // fetch next page
			URL = resp.NextURL
		}
	}
	if uint32(len(subjects)) != expectedTotal {
		err = fmt.Errorf("error fetching PublicSubjects: expected %d, got %d", expectedTotal, len(subjects))
		return
	}
	return
}

// GetPublicSubject gets a subject with the given acronym from the public API
func GetPublicSubject(acronym string) (subject PublicSubject, err error) {
	body, _, err := requestPublic(http.MethodGet, fmt.Sprintf(publicSubjectURLTemplate, acronym))
	if err != nil {
		if err == ErrResourceNotFound {
			err = ErrSubjectNotExists
		}
		err = fmt.Errorf("error getting PublicSubject: %w", err)
		return
	}

	if err = json.Unmarshal(body, &subject); err != nil {
		err = fmt.Errorf("error parsing PublicSubject: %w\n%s", err, string(body))
	}
	return
}

// requestPublic makes a request to Public FIB API using the given HTTP method and URL
func requestPublic(method, URL string) (body []byte, header http.Header, err error) {
	req, err := http.NewRequest(method, URL, nil)
	if err != nil {
		return
	}
	for k, v := range requestHeaders {
		req.Header.Set(k, v)
	}
	req.Header.Set(publicAPIClientIDHeader, publicClientID)

	resp, err := publicClient.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		// API error handling
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest {
			// token has been revoked on server
			err = ErrAuthorizationExpired
		} else if resp.StatusCode == http.StatusNotFound && string(body) == publicSubjectNotFoundResponse {
			err = ErrResourceNotFound
		} else {
			// TODO: handle more other errors
			err = ErrUnknown
		}
	}

	header = resp.Header
	return
}
