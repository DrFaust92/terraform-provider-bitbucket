package bitbucket

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/DrFaust92/bitbucket-go-client"
	"golang.org/x/oauth2"
)

type ClientScopeError struct {
	Error struct {
		Message string `json:"message"`
		Detail  struct {
			Required []string `json:"required"`
			Granted  []string `json:"granted"`
		} `json:"detail"`
	} `json:"error"`
}

func handleClientError(httpResponse *http.Response, err error) error {
	if oauthError, ok := err.(*oauth2.RetrieveError); ok {
		return fmt.Errorf("%s: %s", oauthError.Response.Status, oauthError.ErrorDescription)
	}

	if httpResponse == nil || httpResponse.StatusCode < 400 {
		// Transport errors (timeouts, connection resets) come without a response.
		return err
	}

	clientHttpError, ok := err.(bitbucket.GenericSwaggerError)
	if ok {
		errorBody := extractErrorMessage(clientHttpError.Body())
		return fmt.Errorf("%s: %s", httpResponse.Status, errorBody)
	}

	if err != nil {
		return err
	}

	return nil
}

func extractErrorMessage(body []byte) string {
	var bitbucketHttpError bitbucket.ModelError
	if err := json.Unmarshal(body, &bitbucketHttpError); err == nil && bitbucketHttpError.Error_ != nil && bitbucketHttpError.Error_.Message != "" {
		apiErr := bitbucketHttpError.Error_
		message := apiErr.Message
		if apiErr.Detail != "" {
			message = fmt.Sprintf("%s: %s", message, apiErr.Detail)
		}
		if apiErr.Data != nil {
			if data, err := json.Marshal(apiErr.Data); err == nil {
				message = fmt.Sprintf("%s %s", message, data)
			}
		}
		return message
	}

	var clientScopeErr ClientScopeError
	if err := json.Unmarshal(body, &clientScopeErr); err == nil && clientScopeErr.Error.Message != "" {
		message := clientScopeErr.Error.Message
		required := clientScopeErr.Error.Detail.Required
		granted := clientScopeErr.Error.Detail.Granted
		return fmt.Sprintf("%s Required: %v Granted: %v", message, required, granted)
	}

	return string(body[:])
}
