package core

type SettingsResponse struct {
	Settings         map[string]map[string]any `json:"settings"`
	EditableSections []string                  `json:"editable_sections"`
}

type PatchSettingsRequest struct {
	Settings map[string]map[string]any `json:"settings"`
}

type PatchSettingsResponse struct {
	Status   string                    `json:"status"`
	Settings map[string]map[string]any `json:"settings"`
}
