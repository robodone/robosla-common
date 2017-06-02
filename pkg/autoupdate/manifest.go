package autoupdate

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

const ProdManifestURL = "https://storage.googleapis.com/robosla-agent/robosla-agent.json"

type Manifest struct {
	Version string `json:"version"`
	ARM     string `json:"arm"`
	AMD64   string `json:"amd64"`
}

func (m *Manifest) NeedsUpdate(version string) bool {
	if IsDevBuild(version) {
		// We don't autoupdate dev builds.
		return false
	}
	return version != m.Version
}

func IsDevBuild(version string) bool {
	return version == "dev"
}

func FetchAndParseManifest(manifestURL string) (*Manifest, error) {
	// First, we need to add &t=<something unique> to bypass server-side cache.
	purl, err := url.Parse(manifestURL)
	if err != nil {
		return nil, fmt.Errorf("url.Parse(%q): %v", manifestURL, err)
	}
	query := purl.Query()
	query["t"] = []string{fmt.Sprintf("%d", time.Now().UnixNano())}
	purl.RawQuery = query.Encode()
	manifestURL = purl.String()

	// Now, actually fetch the manifest
	resp, err := http.Get(manifestURL)
	if err != nil {
		return nil, fmt.Errorf("http.Get(%q): %v", manifestURL, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest from %q: %v", manifestURL, err)
	}
	if resp.StatusCode != 200 {
		log.Printf("unexpected HTTP status: %s %d. Trying to parse the manifest anyway.", resp.Status, resp.StatusCode)
	}
	var res Manifest
	if err := json.Unmarshal(body, &res); err != nil {
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("failed to parse manifest json: %v. Most likely it's due to the fact that the HTTP request returned a non-OK status: %s %d", err, resp.Status, resp.StatusCode)
		}
		return nil, fmt.Errorf("failed to parse manifest json: %v", err)
	}
	return &res, nil
}
