package main

import (
	"encoding/json"
	"fmt"
)

// validateInitializeParams checks the client handshake before using the shared
// child's cached result. Protocol versions remain negotiable by the server.
func validateInitializeParams(raw json.RawMessage) error {
	var params struct {
		ProtocolVersion string                     `json:"protocolVersion"`
		Capabilities    map[string]json.RawMessage `json:"capabilities"`
		ClientInfo      *struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"clientInfo"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return fmt.Errorf("initialize params must contain valid protocolVersion, capabilities and clientInfo fields")
	}
	if params.ProtocolVersion == "" {
		return fmt.Errorf("initialize protocolVersion must be a nonempty string")
	}
	if params.Capabilities == nil {
		return fmt.Errorf("initialize capabilities must be an object")
	}
	if params.ClientInfo == nil || params.ClientInfo.Name == "" || params.ClientInfo.Version == "" {
		return fmt.Errorf("initialize clientInfo must contain nonempty name and version strings")
	}
	return nil
}
