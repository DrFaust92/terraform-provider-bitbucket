package bitbucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"golang.org/x/oauth2"
)

// Error represents a error from the bitbucket api.
type Error struct {
	APIError struct {
		Message string `json:"message,omitempty"`
		Key     string `json:"key,omitempty"`
	} `json:"error,omitempty"`
	Message    string `json:"message,omitempty"`
	Key        string `json:"key,omitempty"`
	Type       string `json:"type,omitempty"`
	StatusCode int
	Endpoint   string
}

func (e Error) Error() string {
	return fmt.Sprintf("API Error: %d %s %s", e.StatusCode, e.Endpoint, e.APIError.Message)
}

const (
	// BitbucketEndpoint is the fqdn used to talk to bitbucket
	BitbucketEndpoint string = "https://api.bitbucket.org/"
)

// Client is the base internal Client to talk to bitbuckets API. This should be a username and password
// the password should be a app-password.
type Client struct {
	Username         *string
	Password         *string
	OAuthToken       *string
	OAuthTokenSource oauth2.TokenSource
	HTTPClient       *http.Client
}

// Do Will just call the bitbucket api but also add auth to it and some extra headers
func (c *Client) Do(method, endpoint string, payload *bytes.Buffer, contentType string) (*http.Response, error) {
	absoluteendpoint := BitbucketEndpoint + endpoint
	log.Printf("[DEBUG] Sending request to %s %s", method, absoluteendpoint)

	var bodyreader io.Reader

	if payload != nil {
		log.Printf("[DEBUG] With payload %s", payload.String())
		bodyreader = payload
	}

	req, err := http.NewRequest(method, absoluteendpoint, bodyreader)
	if err != nil {
		return nil, err
	}

	if c.Username != nil && c.Password != nil {
		log.Printf("[DEBUG] Setting Basic Auth")
		req.SetBasicAuth(*c.Username, *c.Password)
	}

	if c.OAuthToken != nil {
		log.Printf("[DEBUG] Setting Bearer Token")
		bearer := "Bearer " + *c.OAuthToken
		req.Header.Add("Authorization", bearer)
	}

	if c.OAuthTokenSource != nil {
		token, err := c.OAuthTokenSource.Token()
		if err != nil {
			return nil, err
		}

		token.SetAuthHeader(req)
	}

	if payload != nil && contentType != "" {
		// Can cause bad request when putting default reviews if set.
		req.Header.Add("Content-Type", contentType)
	}

	req.Close = true

	resp, err := c.HTTPClient.Do(req)
	log.Printf("[DEBUG] Resp: %v Err: %v", resp, err)
	if resp.StatusCode >= 400 || resp.StatusCode < 200 {
		apiError := Error{
			StatusCode: resp.StatusCode,
			Endpoint:   endpoint,
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		log.Printf("[DEBUG] Resp Body: %s", string(body))

		err = json.Unmarshal(body, &apiError)
		if err != nil {
			apiError.APIError.Message = string(body)
		}

		return resp, error(apiError)

	}
	return resp, err
}

// Get is just a helper method to do but with a GET verb
func (c *Client) Get(endpoint string) (*http.Response, error) {
	return c.Do("GET", endpoint, nil, "application/json")
}

// Post is just a helper method to do but with a POST verb
func (c *Client) Post(endpoint string, jsonpayload *bytes.Buffer) (*http.Response, error) {
	return c.Do("POST", endpoint, jsonpayload, "application/json")
}

// PostNonJson is just a helper method to do but with a POST verb without Json Header
func (c *Client) PostNonJson(endpoint string, payload *bytes.Buffer) (*http.Response, error) {
	return c.Do("POST", endpoint, payload, "")
}

// PostWithContentType is just a helper method to do but with a POST verb and a provided content type
func (c *Client) PostWithContentType(endpoint, contentType string, payload *bytes.Buffer) (*http.Response, error) {
	return c.Do("POST", endpoint, payload, contentType)
}

// Put is just a helper method to do but with a PUT verb
func (c *Client) Put(endpoint string, jsonpayload *bytes.Buffer) (*http.Response, error) {
	return c.Do("PUT", endpoint, jsonpayload, "application/json")
}

// PutOnly is just a helper method to do but with a PUT verb and a nil body
func (c *Client) PutOnly(endpoint string) (*http.Response, error) {
	return c.Do("PUT", endpoint, nil, "application/json")
}

// Delete is just a helper to Do but with a DELETE verb
func (c *Client) Delete(endpoint string) (*http.Response, error) {
	return c.Do("DELETE", endpoint, nil, "application/json")
}
