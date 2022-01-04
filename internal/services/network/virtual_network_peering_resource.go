package network

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/azure"

	"github.com/hashicorp/terraform-provider-azurerm/internal/features"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-05-01/network"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/network/parse"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/network/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

// peerMutex is used to prevent multiple Peering resources being created, updated
// or deleted at the same time
var peerMutex = &sync.Mutex{}

func resourceVirtualNetworkPeering() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Create: resourceVirtualNetworkPeeringCreate,
		Read:   resourceVirtualNetworkPeeringRead,
		Update: resourceVirtualNetworkPeeringUpdate,
		Delete: resourceVirtualNetworkPeeringDelete,
		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := parse.VirtualNetworkPeeringID(id)
			return err
		}),

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(30 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(30 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(30 * time.Minute),
		},

		Schema: func() map[string]*pluginsdk.Schema {
			fields := map[string]*pluginsdk.Schema{
				"name": {
					Type:     pluginsdk.TypeString,
					Required: true,
					ForceNew: true,
				},

				"virtual_network_id": {
					Type:         pluginsdk.TypeString,
					Required:     true,
					ForceNew:     true,
					ValidateFunc: validate.VirtualNetworkID,
				},

				"remote_virtual_network_id": {
					Type:         pluginsdk.TypeString,
					Required:     true,
					ForceNew:     true,
					ValidateFunc: validate.VirtualNetworkID,
				},

				"allow_virtual_network_access": {
					Type:     pluginsdk.TypeBool,
					Optional: true,
					Default:  true,
				},

				"allow_forwarded_traffic": {
					Type:     pluginsdk.TypeBool,
					Optional: true,
					Computed: true,
				},

				"allow_gateway_transit": {
					Type:     pluginsdk.TypeBool,
					Optional: true,
					Computed: true,
				},

				"use_remote_gateways": {
					Type:     pluginsdk.TypeBool,
					Optional: true,
					Computed: true,
				},
			}

			if !features.ThreePointOh() {
				fields["resource_group_name"] = &pluginsdk.Schema{
					Type:         pluginsdk.TypeString,
					Optional:     true,
					Computed:     true,
					ForceNew:     true,
					ValidateFunc: azure.ValidateResourceGroupName,
					Deprecated:   "Deprecated in favour of `virtual_network_id`",
					RequiredWith: []string{
						"virtual_network_name",
					},
					ConflictsWith: []string{
						"virtual_network_id",
					},
				}
				fields["virtual_network_name"] = &pluginsdk.Schema{
					Type:       pluginsdk.TypeString,
					Optional:   true,
					Computed:   true,
					ForceNew:   true,
					Deprecated: "Deprecated in favour of `virtual_network_id`",
					RequiredWith: []string{
						"resource_group_name",
					},
					ConflictsWith: []string{
						"virtual_network_id",
					},
				}

				fields["virtual_network_id"] = &schema.Schema{
					Type:         pluginsdk.TypeString,
					Optional:     true,
					Computed:     true,
					ForceNew:     true,
					ValidateFunc: validate.VirtualNetworkID,
					ConflictsWith: []string{
						"resource_group_name",
						"virtual_network_name",
					},
				}
			}

			return fields
		}(),
	}
}

func resourceVirtualNetworkPeeringCreate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Network.VnetPeeringsClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	virtualNetworkIdRaw := d.Get("virtual_network_id").(string)
	if virtualNetworkIdRaw == "" {
		// TODO: remove in 3.0
		subscriptionId := meta.(*clients.Client).Account.SubscriptionId
		virtualNetworkIdRaw = parse.NewVirtualNetworkID(subscriptionId, d.Get("resource_group_name").(string), d.Get("virtual_network_name").(string)).ID()
	}
	virtualNetworkId, err := parse.VirtualNetworkID(virtualNetworkIdRaw)
	if err != nil {
		return err
	}

	id := parse.NewVirtualNetworkPeeringID(virtualNetworkId.SubscriptionId, virtualNetworkId.ResourceGroup, virtualNetworkId.Name, d.Get("name").(string))
	existing, err := client.Get(ctx, id.ResourceGroup, id.VirtualNetworkName, id.Name)
	if err != nil {
		if !utils.ResponseWasNotFound(existing.Response) {
			return fmt.Errorf("checking for presence of existing %s: %+v", id, err)
		}
	}

	if !utils.ResponseWasNotFound(existing.Response) {
		return tf.ImportAsExistsError("azurerm_virtual_network_peering", id.ID())
	}

	model := network.VirtualNetworkPeering{
		VirtualNetworkPeeringPropertiesFormat: &network.VirtualNetworkPeeringPropertiesFormat{
			AllowVirtualNetworkAccess: utils.Bool(d.Get("allow_virtual_network_access").(bool)),
			AllowForwardedTraffic:     utils.Bool(d.Get("allow_forwarded_traffic").(bool)),
			AllowGatewayTransit:       utils.Bool(d.Get("allow_gateway_transit").(bool)),
			UseRemoteGateways:         utils.Bool(d.Get("use_remote_gateways").(bool)),
			RemoteVirtualNetwork: &network.SubResource{
				ID: utils.String(d.Get("remote_virtual_network_id").(string)),
			},
		},
	}

	peerMutex.Lock()
	defer peerMutex.Unlock()

	timeout, _ := ctx.Deadline()
	stateConf := &pluginsdk.StateChangeConf{
		Pending:    []string{"Pending"},
		Target:     []string{"Succeeded"},
		Refresh:    virtualNetworkPeeringCreateFunc(ctx, client, id, model),
		MinTimeout: 15 * time.Second,
		Timeout:    time.Until(timeout),
	}
	if _, err = stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for creation of %s: %+v", id, err)
	}

	d.SetId(id.ID())
	return resourceVirtualNetworkPeeringRead(d, meta)
}

func resourceVirtualNetworkPeeringRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Network.VnetPeeringsClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.VirtualNetworkPeeringID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, id.ResourceGroup, id.VirtualNetworkName, id.Name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[INFO] %s was not found - removing from state", *id)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("retrieving %s: %+v", *id, err)
	}

	// update appropriate values
	d.Set("name", id.Name)
	d.Set("virtual_network_id", parse.NewVirtualNetworkID(id.SubscriptionId, id.ResourceGroup, id.VirtualNetworkName).ID())

	if !features.ThreePointOh() {
		d.Set("resource_group_name", id.ResourceGroup)
		d.Set("virtual_network_name", id.VirtualNetworkName)
	}

	if peer := resp.VirtualNetworkPeeringPropertiesFormat; peer != nil {
		d.Set("allow_virtual_network_access", peer.AllowVirtualNetworkAccess)
		d.Set("allow_forwarded_traffic", peer.AllowForwardedTraffic)
		d.Set("allow_gateway_transit", peer.AllowGatewayTransit)
		d.Set("use_remote_gateways", peer.UseRemoteGateways)

		remoteVirtualNetworkId := ""
		if peer.RemoteVirtualNetwork != nil && peer.RemoteVirtualNetwork.ID != nil {
			parsed, err := parse.VirtualNetworkIDInsensitively(*peer.RemoteVirtualNetwork.ID)
			if err != nil {
				return fmt.Errorf("parsing %q as a virtual network id: %+v", *peer.RemoteVirtualNetwork.ID, err)
			}
			remoteVirtualNetworkId = parsed.ID()
		}
		d.Set("remote_virtual_network_id", remoteVirtualNetworkId)
	}

	return nil
}

func resourceVirtualNetworkPeeringUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Network.VnetPeeringsClient
	ctx, cancel := timeouts.ForUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.VirtualNetworkPeeringID(d.Id())
	if err != nil {
		return err
	}

	peerMutex.Lock()
	defer peerMutex.Unlock()

	existing, err := client.Get(ctx, id.ResourceGroup, id.VirtualNetworkName, id.Name)
	if err != nil {
		return fmt.Errorf("retrieving %s: %+v", *id, err)
	}
	if existing.VirtualNetworkPeeringPropertiesFormat == nil {
		return fmt.Errorf("retrieving %s: `properties` was nil", *id)
	}

	props := *existing.VirtualNetworkPeeringPropertiesFormat

	if d.HasChange("allow_forwarded_traffic") {
		props.AllowForwardedTraffic = utils.Bool(d.Get("allow_forwarded_traffic").(bool))
	}

	if d.HasChange("allow_gateway_transit") {
		props.AllowGatewayTransit = utils.Bool(d.Get("allow_gateway_transit").(bool))
	}

	if d.HasChange("allow_virtual_network_access") {
		props.AllowVirtualNetworkAccess = utils.Bool(d.Get("allow_virtual_network_access").(bool))
	}

	if d.HasChange("use_remote_gateways") {
		props.UseRemoteGateways = utils.Bool(d.Get("use_remote_gateways").(bool))
	}

	model := network.VirtualNetworkPeering{
		VirtualNetworkPeeringPropertiesFormat: &props,
	}

	future, err := client.CreateOrUpdate(ctx, id.ResourceGroup, id.VirtualNetworkName, id.Name, model, network.SyncRemoteAddressSpaceTrue)
	if err != nil {
		return fmt.Errorf("updating %s: %+v", *id, err)
	}

	if err := future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("updating %s: %+v", *id, err)
	}

	return resourceVirtualNetworkPeeringRead(d, meta)
}

func resourceVirtualNetworkPeeringDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Network.VnetPeeringsClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.VirtualNetworkPeeringID(d.Id())
	if err != nil {
		return err
	}

	peerMutex.Lock()
	defer peerMutex.Unlock()

	future, err := client.Delete(ctx, id.ResourceGroup, id.VirtualNetworkName, id.Name)
	if err != nil {
		return fmt.Errorf("deleting %s: %+v", *id, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("waiting for deletion of %s: %+v", *id, err)
	}

	return err
}

func virtualNetworkPeeringCreateFunc(ctx context.Context, client *network.VirtualNetworkPeeringsClient, id parse.VirtualNetworkPeeringId, model network.VirtualNetworkPeering) resource.StateRefreshFunc {
	return func() (result interface{}, state string, err error) {
		future, err := client.CreateOrUpdate(ctx, id.ResourceGroup, id.VirtualNetworkName, id.Name, model, network.SyncRemoteAddressSpaceTrue)
		if err != nil {
			if utils.ResponseErrorIsRetryable(err) {
				return "Pending", "Pending", err
			} else if future.Response() != nil && future.Response().StatusCode == 400 && strings.Contains(err.Error(), "ReferencedResourceNotProvisioned") {
				// Resource is not yet ready, this may be the case if the Vnet was just created or another peering was just initiated.
				return "Pending", "Pending", err
			}

			return "Failed", "Failed", err
		}

		if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
			return "Failure", "Failure", err
		}

		return "Succeeded", "Succeeded", nil
	}
}
