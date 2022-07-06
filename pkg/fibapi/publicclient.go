package fibapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GetPublicSubject gets a subject with the given acronym from the public API
func GetPublicSubject(acronym string) (subject PublicSubject, err error) {
	body, _, err := requestPublic(http.MethodGet, fmt.Sprintf(PublicSubjectURLTemplate, acronym))
	if err != nil {
		if err == ErrResourceNotFound && string(body) == PublicSubjectNotFoundResponse {
			err = ErrSubjectNotExists
		}
		return
	}

	err = json.Unmarshal(body, &subject)
	if err != nil {
		err = fmt.Errorf("error parsing PublicSubject response: %s\n%s", string(body), err)
	}
	return
}

// requestPublic makes a request to FIB Public API with the given method and URL
func requestPublic(method, URL string) (body []byte, header http.Header, err error) {
	req, err := http.NewRequest(method, URL, nil)
	if err != nil {
		return
	}
	req.Header.Set("client_id", publicClientID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	header = resp.Header
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
