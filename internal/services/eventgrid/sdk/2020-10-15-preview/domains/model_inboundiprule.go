package domains

type InboundIpRule struct {
	Action *IpActionType `json:"action,omitempty"`
	IpMask *string       `json:"ipMask,omitempty"`
}
