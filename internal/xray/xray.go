package xray

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	repo            = "XTLS/Xray-core"
	fallbackVersion = "v26.6.1"
)

func binName() string {
	if runtime.GOOS == "windows" {
		return "xray.exe"
	}
	return "xray"
}

func binDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "alexandria", "bin"), nil
}

func Locate() string {
	name := binName()
	if exe, err := os.Executable(); err == nil {
		if p := filepath.Join(filepath.Dir(exe), name); isExec(p) {
			return p
		}
	}
	if d, err := binDir(); err == nil {
		if p := filepath.Join(d, name); isExec(p) {
			return p
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return ""
}

func isExec(p string) bool {
	fi, err := os.Stat(p)
	if err != nil || fi.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return fi.Mode()&0111 != 0
}

func Ensure(ctx context.Context) (string, error) {
	if p := Locate(); p != "" {
		return p, nil
	}
	return download(ctx)
}

func download(ctx context.Context) (string, error) {
	asset, err := assetName()
	if err != nil {
		return "", err
	}
	dir, err := binDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}

	zipURL, dgstURL := resolveURLs(ctx, asset)

	tmp, err := os.CreateTemp(dir, "xray-*.zip")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())

	h := sha256.New()
	if err := get(ctx, zipURL, io.MultiWriter(tmp, h)); err != nil {
		tmp.Close()
		return "", err
	}
	tmp.Close()

	if dgstURL == "" {
		return "", errors.New("no checksum available for release asset")
	}
	want, err := fetchSHA256(ctx, dgstURL)
	if err != nil {
		return "", fmt.Errorf("verify checksum: %w", err)
	}
	if got := hex.EncodeToString(h.Sum(nil)); !strings.EqualFold(got, want) {
		return "", fmt.Errorf("checksum mismatch: got %s want %s", got, want)
	}

	if err := extract(tmp.Name(), dir); err != nil {
		return "", err
	}
	dest := filepath.Join(dir, binName())
	if !isExec(dest) {
		return "", errors.New("xray binary not found in archive")
	}
	return dest, nil
}

func resolveURLs(ctx context.Context, asset string) (zipURL, dgstURL string) {
	if rels, err := listReleases(ctx); err == nil {
		for _, r := range rels {
			if r.Draft {
				continue
			}
			var z, d string
			for _, a := range r.Assets {
				switch a.Name {
				case asset:
					z = a.URL
				case asset + ".dgst":
					d = a.URL
				}
			}
			if z != "" {
				return z, d
			}
		}
	}
	base := "https://github.com/" + repo + "/releases/download/" + fallbackVersion + "/"
	return base + asset, base + asset + ".dgst"
}

type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type ghRelease struct {
	Tag    string    `json:"tag_name"`
	Draft  bool      `json:"draft"`
	Assets []ghAsset `json:"assets"`
}

func listReleases(ctx context.Context) ([]ghRelease, error) {
	url := "https://api.github.com/repos/" + repo + "/releases?per_page=15"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api: %s", resp.Status)
	}
	var rels []ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rels); err != nil {
		return nil, err
	}
	return rels, nil
}

func get(ctx context.Context, url string, w io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: %s", url, resp.Status)
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

func fetchSHA256(ctx context.Context, url string) (string, error) {
	var b strings.Builder
	if err := get(ctx, url, &b); err != nil {
		return "", err
	}
	for _, line := range strings.Split(b.String(), "\n") {
		norm := strings.ReplaceAll(strings.ToUpper(line), "-", "")
		if !strings.Contains(norm, "SHA256") {
			continue
		}
		for _, f := range strings.FieldsFunc(line, func(r rune) bool { return r == ' ' || r == '\t' || r == '=' || r == ':' }) {
			if isHex64(f) {
				return f, nil
			}
		}
	}
	return "", errors.New("sha256 not found in digest")
}

func isHex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func extract(zipPath, dir string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()

	for _, f := range zr.File {
		name := filepath.Base(f.Name)
		if name != binName() && !strings.HasSuffix(name, ".dat") {
			continue
		}
		mode := os.FileMode(0644)
		if name == binName() {
			mode = 0755
		}
		if err := writeZipFile(f, filepath.Join(dir, name), mode); err != nil {
			return err
		}
	}
	return nil
}

func writeZipFile(f *zip.File, dest string, mode os.FileMode) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	return err
}

func assetName() (string, error) {
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return "Xray-linux-64.zip", nil
		case "arm64":
			return "Xray-linux-arm64-v8a.zip", nil
		case "386":
			return "Xray-linux-32.zip", nil
		case "arm":
			return "Xray-linux-arm32-v7a.zip", nil
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			return "Xray-macos-64.zip", nil
		case "arm64":
			return "Xray-macos-arm64-v8a.zip", nil
		}
	case "windows":
		switch runtime.GOARCH {
		case "amd64":
			return "Xray-windows-64.zip", nil
		case "arm64":
			return "Xray-windows-arm64-v8a.zip", nil
		case "386":
			return "Xray-windows-32.zip", nil
		}
	}
	return "", fmt.Errorf("unsupported platform %s/%s", runtime.GOOS, runtime.GOARCH)
}
