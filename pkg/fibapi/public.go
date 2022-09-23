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
	var expectedTotal uint32

	URL := PublicSubjectsURL
	for {
		body, _, err := requestPublic(http.MethodGet, URL)
		if err != nil {
			return nil, err
		}

		var resp PublicSubjectsResponse
		if err = json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("error parsing PublicSubjects response: %s\n%s", string(body), err)
		}

		expectedTotal = resp.Count
		subjects = append(subjects, resp.Results...)
		if resp.NextURL == "" { // all fetched
			break
		} else { // fetch next page
			URL = resp.NextURL
		}
	}
	if uint32(len(subjects)) != expectedTotal {
		return nil, fmt.Errorf("error fetching PublicSubjects: expected %d, got %d", expectedTotal, len(subjects))
	}
	return
}

// GetPublicSubject gets a subject with the given acronym from the public API
func GetPublicSubject(acronym string) (subject PublicSubject, err error) {
	body, _, err := requestPublic(http.MethodGet, fmt.Sprintf(PublicSubjectURLTemplate, acronym))
	if err != nil {
		if err == ErrResourceNotFound && string(body) == PublicSubjectNotFoundResponse {
			err = ErrSubjectNotExists
		}
		return
	}

	if err = json.Unmarshal(body, &subject); err != nil {
		err = fmt.Errorf("error parsing PublicSubject response: %s\n%s", string(body), err)
	}
	return
}

const clientIDHeader = "client_id"

// requestPublic makes a request to FIB Public API with the given method and URL
func requestPublic(method, URL string) (body []byte, header http.Header, err error) {
	req, err := http.NewRequest(method, URL, nil)
	if err != nil {
		return
	}
	for k, v := range requestHeaders {
		req.Header.Set(k, v)
	}
	req.Header.Set(clientIDHeader, publicClientID)

	resp, err := publicClient.Do(req)
	if err != nil {
		return
	}

	header = resp.Header
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest {
			// token has been revoked on server
			err = ErrAuthorizationExpired
		} else if resp.StatusCode == http.StatusNotFound {
			err = ErrResourceNotFound
		} else {
			// TODO: handle more other errors
			err = ErrUnknown
		}
	}
	return
}
