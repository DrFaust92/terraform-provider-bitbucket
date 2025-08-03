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
		return nil
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
	if err := json.Unmarshal(body, &bitbucketHttpError); err == nil {
		return bitbucketHttpError.Error_.Message
	}

	var clientScopeErr ClientScopeError
	if err := json.Unmarshal(body, &clientScopeErr); err == nil {
		message := clientScopeErr.Error.Message
		required := clientScopeErr.Error.Detail.Required
		granted := clientScopeErr.Error.Detail.Granted
		return fmt.Sprintf("%s Required: %v Granted: %v", message, required, granted)
	}

	return string(body[:])
}
