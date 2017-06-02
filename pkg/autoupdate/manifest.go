package autoupdate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
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

func (m *Manifest) BinaryURLArch(arch string) (string, error) {
	var res string
	switch arch {
	case "arm":
		res = m.ARM
	case "amd64":
		res = m.AMD64
	default:
		return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}
	return res, nil
}

func (m *Manifest) BinaryURL() (string, error) {
	return m.BinaryURLArch(runtime.GOARCH)
}

func (m *Manifest) DownloadBinaryArch(arch string) ([]byte, error) {
	binaryURL, err := m.BinaryURLArch(arch)
	if err != nil {
		return nil, err
	}
	resp, err := http.Get(binaryURL)
	if err != nil {
		return nil, fmt.Errorf("http.Get(%q): %v", binaryURL, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch a new binary from %q: %v", binaryURL, err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected HTTP status: %s %d.", resp.Status, resp.StatusCode)
	}
	return body, nil
}

func (m *Manifest) QualifyBinary(binaryPath string) error {
	data, err := exec.Command(binaryPath, "--version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run %s --version, err: %v\ncombined output:\n%s",
			binaryPath, err, string(data))
	}
	binaryVersion := strings.TrimSpace(string(data))
	if m.Version != binaryVersion {
		return fmt.Errorf("version mismatch. Want: %q, got: %q", m.Version, binaryVersion)
	}
	// While it's certainly not a comprehensive test, at least we know that the binary
	// can run on the current system and that the version matches.
	// TODO(krasin): check binary hash (need to put it into the manifest first)
	// TODO(krasin): check that the binary can access network.
	return nil
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
		if resp.StatusCode >= 400 {
			// Definitely not good
			return nil, fmt.Errorf("HTTP error while fetching a manifest from %q: %s %d", manifestURL, resp.Status, resp.StatusCode)
		}
		log.Printf("unexpected HTTP status: %s %d. Trying to parse the manifest anyway.", resp.Status, resp.StatusCode)
	}
	var res Manifest
	if err := json.Unmarshal(body, &res); err != nil {
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("failed to parse manifest json: %v. Most likely it's due to the fact that the HTTP request returned a non-OK status: %s %d", err, resp.Status, resp.StatusCode)
		}
		return nil, fmt.Errorf("failed to parse manifest json: %v", err)
	}
	if res.Version == "" {
		return nil, errors.New("invalid manifest: missing version")
	}
	return &res, nil
}

func UpdateCurrentBinaryIfNeeded(manifestURL, version string) (needsRestart bool, err error) {
	// Fetch manifest
	m, err := FetchAndParseManifest(manifestURL)
	if err != nil {
		return false, fmt.Errorf("FetchAndParseManifest(%q): %v", manifestURL, err)
	}
	// Check, if we need to update the currently running binary.
	if !m.NeedsUpdate(version) {
		return false, nil
	}
	// Fetch the binary.
	binary, err := m.DownloadBinaryArch(runtime.GOARCH)
	// Detecting currently running binary.
	curBinaryPath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("failed to get the path to the currently running executable: %v", err)
	}
	log.Printf("Current executable path: %s\n", curBinaryPath)
	newBinaryPath := curBinaryPath + ".new"
	// Save the new binary.
	if err := ioutil.WriteFile(newBinaryPath, binary, 0755); err != nil {
		return false, fmt.Errorf("failed to save the new binary: %v", err)
	}
	// Check the new binary version (as well as the ability to run on the current computer).
	if err := m.QualifyBinary(newBinaryPath); err != nil {
		return false, fmt.Errorf("failed to quality the new binary %s: %v", newBinaryPath, err)
	}
	// Rename current binary to old.
	oldBinaryPath := curBinaryPath + ".old"
	if err := os.Rename(curBinaryPath, oldBinaryPath); err != nil {
		return false, fmt.Errorf("failed to rename current binary (%s) into the old one (%s): %v",
			curBinaryPath, oldBinaryPath, err)
	}
	// Rename new binary to the current.
	if err := os.Rename(newBinaryPath, curBinaryPath); err != nil {
		// Make best effort to rename old binary to the current one.
		os.Rename(oldBinaryPath, curBinaryPath)
		return false, fmt.Errorf("failed to rename new binary (%s) into the current one (%s): %v",
			newBinaryPath, curBinaryPath, err)
	}
	// We have updated the binary and need to restart.
	return true, nil
}

func GetLatestProdBinaryArch(arch string) ([]byte, error) {
	m, err := FetchAndParseManifest(ProdManifestURL)
	if err != nil {
		return nil, err
	}
	return m.DownloadBinaryArch(arch)
}
