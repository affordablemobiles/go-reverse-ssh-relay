package shared

type JSONType struct {
	MType string `json:"type"`
}

type JSONIdentityNotify struct {
	MType    string `json:"type"`
	Project  string `json:"project"`
	Service  string `json:"service"`
	Version  string `json:"version"`
	Instance string `json:"instance"`
}

type JSONIdentityConfirm struct {
	MType    string `json:"type"`
	Accepted bool   `json:"accepted"`
}
