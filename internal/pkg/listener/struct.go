package listener

import "strings"

type Oslo struct {
	Version string `json:"oslo.version"`
	Message string `json:"oslo.message"`
}

type Message struct {
	MessageID string                 `json:"message_id"`
	EventType string                 `json:"event_type"`
	Priority  string                 `json:"priority"`
	Payload   map[string]interface{} `json:"payload"`
}

func (m *Message) getTunnelMetaData() []string {
	metadata := m.Payload["metadata"].(map[string]interface{})
	if metadata["tunnel"] != nil {
		return strings.Split(metadata["tunnel"].(string), ",")
	}

	return nil
}

func (m *Message) getInstanceName() (string, string) {
	instanceID := m.Payload["instance_id"].(string)
	instanceName := m.Payload["display_name"].(string)
	return instanceID, instanceName
}

func (m *Message) isActive() bool {
	return m.Payload["state"] == ActiveStatus
}
