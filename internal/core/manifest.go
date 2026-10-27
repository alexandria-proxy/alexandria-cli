package core

import (
	_ "embed"
	"encoding/json"
)

//go:embed manifest.json
var manifest []byte

func Versions() string {
	var m struct {
		Version string `json:"version"`
		Singbox struct {
			Version string `json:"version"`
		} `json:"singbox"`
	}
	if json.Unmarshal(manifest, &m) != nil || m.Version == "" {
		return "xray-core unknown"
	}
	line := "xray-core " + m.Version
	if m.Singbox.Version != "" {
		line += ", sing-box " + m.Singbox.Version
	}
	return line
}
