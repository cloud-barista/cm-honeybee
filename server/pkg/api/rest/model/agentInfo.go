package model

type AgentInfo struct {
  Connection string `json:"connection_name"`
  Result     string `json:"result"`
  ErrorMsg   error  `json:"error_msg,omitempty"`
}
