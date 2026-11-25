package bitbucket

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// groupsWorkspace returns the workspace to use in 1.0/groups API paths. That
// API needs the workspace slug: with a workspace UUID, such as the id of the
// bitbucket_workspace data source, Bitbucket answers 405 or 403. A UUID is
// resolved to the slug through the 2.0 workspaces API; a slug is returned as
// is.
func groupsWorkspace(client Client, workspace string) (string, error) {
	if !strings.HasPrefix(workspace, "{") {
		return workspace, nil
	}

	res, err := client.Get("2.0/workspaces/" + url.PathEscape(workspace))
	if err != nil {
		return "", fmt.Errorf("resolving workspace %s to its slug: %w", workspace, err)
	}
	defer res.Body.Close()

	var ws struct {
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(res.Body).Decode(&ws); err != nil {
		return "", fmt.Errorf("resolving workspace %s to its slug: %w", workspace, err)
	}
	if ws.Slug == "" {
		return "", fmt.Errorf("resolving workspace %s to its slug: the response has no slug", workspace)
	}
	return ws.Slug, nil
}
