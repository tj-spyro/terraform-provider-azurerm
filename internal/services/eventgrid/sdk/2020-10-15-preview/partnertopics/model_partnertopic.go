package partnertopics

import (
	"github.com/hashicorp/terraform-provider-azurerm/internal/identity"
)

type PartnerTopic struct {
	Id         *string                                 `json:"id,omitempty"`
	Identity   *identity.SystemUserAssignedIdentityMap `json:"identity,omitempty"`
	Location   string                                  `json:"location"`
	Name       *string                                 `json:"name,omitempty"`
	Properties *PartnerTopicProperties                 `json:"properties,omitempty"`
	SystemData *SystemData                             `json:"systemData,omitempty"`
	Tags       *map[string]string                      `json:"tags,omitempty"`
	Type       *string                                 `json:"type,omitempty"`
}
