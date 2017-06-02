package autoupdate

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

const ProdManifestURL = "https://storage.googleapis.com/robosla-agent/robosla-agent.json"

type Manifest struct {
	ARM   string `json:"arm"`
	AMD64 string `json:"amd64"`
}

func FetchAndParseManifest(url string) (*Manifest, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http.Get(%q): %v", url, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest from %q: %v", url, err)
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
