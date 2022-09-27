package fibapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	publicClient   *http.Client
	publicClientID string
)

// GetPublicSubjects gets all subjects from the public API
func GetPublicSubjects() ([]PublicSubject, error) {
	deadline := 20 * time.Second

	var saidTotal uint32
	var subjects []PublicSubject

	URL := publicSubjectsURL
	start := time.Now()
	for { // loop until all pages are fetched
		if time.Since(start) > deadline {
			return nil, fmt.Errorf("fibapi: error fetching PublicSubjects: deadline exceeded")
		}

		body, _, err := requestPublic(http.MethodGet, URL)
		if err != nil {
			return nil, err
		}
		var resp PublicSubjectsResponse
		if err = json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("fibapi: error parsing PublicSubjects: %w", err)
		}

		if saidTotal == 0 {
			saidTotal = resp.Count
			subjects = make([]PublicSubject, 0, saidTotal)
		} else if resp.Count != saidTotal {
			return nil, fmt.Errorf("fibapi: error fetching PublicSubjects: said total changed during fetching")
		}
		subjects = append(subjects, resp.Results...)

		if resp.NextURL == "" { // all fetched
			break
		}
		URL = resp.NextURL // continue to fetch the next page
	}
	if uint32(len(subjects)) != saidTotal {
		return nil, fmt.Errorf("fibapi: error fetching PublicSubjects: said total %d, got %d", saidTotal, len(subjects))
	}
	return subjects, nil
}

// GetPublicSubject gets a subject with the given acronym from the public API
func GetPublicSubject(acronym string) (subject PublicSubject, err error) {
	body, _, err := requestPublic(http.MethodGet, fmt.Sprintf(publicSubjectURLTemplate, acronym))
	if err != nil {
		return
	}
	if err = json.Unmarshal(body, &subject); err != nil {
		err = fmt.Errorf("fibapi: error parsing PublicSubject: %w", err)
	}
	return
}

// requestPublic makes a request to Public FIB API using the given HTTP method and URL
func requestPublic(method, URL string) (body []byte, header http.Header, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, URL, nil)
	if err != nil {
		err = fmt.Errorf("fibapi: error creating request: %w", err)
		return
	}
	for k, v := range requestHeaders {
		req.Header.Set(k, v)
	}
	req.Header.Set(publicAPIClientIDHeader, publicClientID)

	resp, err := publicClient.Do(req)
	if err != nil {
		err = fmt.Errorf("fibapi: error making request: %w", err)
		return
	}

	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("fibapi: error reading response body: %w", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		// API error handling
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest {
			// token has expired or has been revoked on server
			err = ErrAuthorizationExpired
		} else if resp.StatusCode == http.StatusNotFound {
			var r Response
			if err = json.Unmarshal(body, &r); err != nil {
				err = fmt.Errorf("fibapi: error parsing response: %w", err)
				return
			}
			if r.Detail == resourceNotFoundResponseDetail {
				err = ErrResourceNotFound
			} else {
				if r.Detail == "" {
					r.Detail = "(no detail message)"
				}
				err = fmt.Errorf("fibapi: error in response: %s", r.Detail)
			}
		} else {
			err = fmt.Errorf("fibapi: bad response (%d): %s", resp.StatusCode, string(body))
		}
	}

	header = resp.Header
	return
}
