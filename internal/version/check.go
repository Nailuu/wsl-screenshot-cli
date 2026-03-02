package version

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// apiURL is the GitHub releases API endpoint, declared as a var so tests can
// override it with an httptest server (same pattern as internal/poller).
var apiURL = "https://api.github.com/repos/Nailuu/wsl-screenshot-cli/releases/latest"

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// CheckForUpdate queries the GitHub releases API and returns the latest
// version string if it is newer than currentVersion. Returns "" if up to date.
// Skips the check entirely for dev builds. All errors are returned to the
// caller so the command layer can silently ignore them.
func CheckForUpdate(currentVersion string) (string, error) {
	if currentVersion == "dev" {
		return "", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	latest := strings.TrimPrefix(release.TagName, "v")

	newer, err := isNewer(latest, currentVersion)
	if err != nil {
		return "", err
	}
	if newer {
		return latest, nil
	}
	return "", nil
}

// parseSemver splits a "major.minor.patch" string into three integers.
func parseSemver(v string) (major, minor, patch int, err error) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid semver: %q", v)
	}
	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major version: %w", err)
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor version: %w", err)
	}
	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid patch version: %w", err)
	}
	return major, minor, patch, nil
}

// isNewer returns true if latest is strictly newer than current.
func isNewer(latest, current string) (bool, error) {
	lMaj, lMin, lPat, err := parseSemver(latest)
	if err != nil {
		return false, fmt.Errorf("latest: %w", err)
	}
	cMaj, cMin, cPat, err := parseSemver(current)
	if err != nil {
		return false, fmt.Errorf("current: %w", err)
	}

	if lMaj != cMaj {
		return lMaj > cMaj, nil
	}
	if lMin != cMin {
		return lMin > cMin, nil
	}
	return lPat > cPat, nil
}
