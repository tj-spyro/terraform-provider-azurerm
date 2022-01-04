---
subcategory: "Network"
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_virtual_network_peering"
description: |-
  Managed a Virtual Network Peering, which can be used to access resources within the Linked Virtual Network.
---

# azurerm_virtual_network_peering

Managed a Virtual Network Peering, which can be used to access resources within the Linked Virtual Network.

## Example Usage

```hcl
resource "azurerm_resource_group" "example" {
  name     = "example-resources"
  location = "West Europe"
}

resource "azurerm_virtual_network" "first" {
  name                = "first-network"
  resource_group_name = azurerm_resource_group.example.name
  location            = azurerm_resource_group.example.location
  address_space       = ["10.0.1.0/24"]
}

resource "azurerm_virtual_network" "second" {
  name                = "second-network"
  resource_group_name = azurerm_resource_group.example.name
  location            = azurerm_resource_group.example.location
  address_space       = ["10.0.2.0/24"]
}

resource "azurerm_virtual_network_peering" "first-to-second" {
  name                      = "first-to-second"
  resource_group_name       = azurerm_resource_group.example.name
  virtual_network_id        = azurerm_virtual_network.first.id
  remote_virtual_network_id = azurerm_virtual_network.second.id
}

resource "azurerm_virtual_network_peering" "second-to-first" {
  name                      = "second-to-first"
  resource_group_name       = azurerm_resource_group.example.name
  virtual_network_id        = azurerm_virtual_network.second.id
  remote_virtual_network_id = azurerm_virtual_network.first.id
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name which should be used for this Virtual Network Peering. Changing this forces a new resource to be created.

* `virtual_network_id` - (Optional) The ID of the (Local) Virtual Network. Changing this forces a new resource to be created.

* `remote_virtual_network_id` - (Required) The ID of the Remote Virtual Network.  Changing this forces a new resource to be created.

---

* `allow_forwarded_traffic` - (Optional) Can traffic be forwarded from the Remote Virtual Network? Defaults to `false`.

* `allow_gateway_transit` - (Optional) Can Gateway Links be used in the Remote Virtual Network to link to the (Local) Virtual Network? Defaults to `false`.

* `allow_virtual_network_access` - (Optional) Can Resources within the Remote Virtual Network access items in the (Local) Virtual Network? Defaults to `true`.

* `use_remote_gateways` - (Optional) Can Remote Gateways be used on the Local Virtual Network? Defaults to `false`.

-> **Note:** When this field is set to `true` the Remote Virtual Network must have `allow_gateway_transit` set to `true`.

-> **Note:** This field must be set to `false` when creating a Global Virtual Network Peering.

~> **Note:** Only one Peering within a Virtual Network can have a Gateway (set to `true`).

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the Virtual Network Peering.

## Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/docs/configuration/resources.html#timeouts) for certain actions:

* `create` - (Defaults to 30 minutes) Used when creating the Virtual Network Peering.
* `update` - (Defaults to 30 minutes) Used when updating the Virtual Network Peering.
* `read` - (Defaults to 5 minutes) Used when retrieving the Virtual Network Peering.
* `delete` - (Defaults to 30 minutes) Used when deleting the Virtual Network Peering.

## Import

Virtual Network Peerings can be imported using the `resource id`, e.g.

```shell
terraform import azurerm_virtual_network_peering.example /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-resource-group/providers/Microsoft.Network/virtualNetworks/example-network/virtualNetworkPeerings/example-peering
```
