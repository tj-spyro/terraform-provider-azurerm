package containers

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2021-08-01/containerservice"
	"github.com/Azure/go-autorest/autorest/azure"
	commonValidate "github.com/hashicorp/terraform-provider-azurerm/helpers/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/features"
	containerValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/containers/validate"
	laparse "github.com/hashicorp/terraform-provider-azurerm/internal/services/loganalytics/parse"
	logAnalyticsValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/loganalytics/validate"
	applicationGatewayValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/network/validate"
	subnetValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/network/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

const (
	// note: the casing on these keys is important
	aciConnectorKey                 = "aciConnectorLinux"
	azurePolicyKey                  = "azurepolicy"
	kubernetesDashboardKey          = "kubeDashboard"
	httpApplicationRoutingKey       = "httpApplicationRouting"
	omsAgentKey                     = "omsagent"
	ingressApplicationGatewayKey    = "ingressApplicationGateway"
	openServiceMeshKey              = "openServiceMesh"
	azureKeyvaultSecretsProviderKey = "azureKeyvaultSecretsProvider"
)

// The AKS API hard-codes which add-ons are supported in which environment
// as such unfortunately we can't just send "disabled" - we need to strip
// the unsupported addons from the HTTP response. As such this defines
// the list of unsupported addons in the defined region - e.g. by being
// omitted from this list an addon/environment combination will be supported
var unsupportedAddonsForEnvironment = map[string][]string{
	azure.ChinaCloud.Name: {
		aciConnectorKey,                 // https://github.com/hashicorp/terraform-provider-azurerm/issues/5510
		httpApplicationRoutingKey,       // https://github.com/hashicorp/terraform-provider-azurerm/issues/5960
		kubernetesDashboardKey,          // https://github.com/hashicorp/terraform-provider-azurerm/issues/7487
		openServiceMeshKey,              // Preview features are not supported in Azure China
		azureKeyvaultSecretsProviderKey, // Preview features are not supported in Azure China
	},
	azure.USGovernmentCloud.Name: {
		httpApplicationRoutingKey,       // https://github.com/hashicorp/terraform-provider-azurerm/issues/5960
		kubernetesDashboardKey,          // https://github.com/hashicorp/terraform-provider-azurerm/issues/7136
		openServiceMeshKey,              // Preview features are not supported in Azure Government
		azureKeyvaultSecretsProviderKey, // Preview features are not supported in Azure China
	},
}

func schemaKubernetesAddOnProfiles() *pluginsdk.Schema {
	// TODO 3.0 - Remove this block
	if !features.ThreePointOh() {
		return &pluginsdk.Schema{
			Type:     pluginsdk.TypeList,
			MaxItems: 1,
			Optional: true,
			Computed: true,
			Elem: &pluginsdk.Resource{
				Schema: map[string]*pluginsdk.Schema{
					"aci_connector_linux": {
						Type:     pluginsdk.TypeList,
						MaxItems: 1,
						Optional: true,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"enabled": {
									Type:     pluginsdk.TypeBool,
									Required: true,
								},

								"subnet_name": {
									Type:         pluginsdk.TypeString,
									Optional:     true,
									ValidateFunc: validation.StringIsNotEmpty,
								},
							},
						},
					},

					"azure_policy": {
						Type:     pluginsdk.TypeList,
						MaxItems: 1,
						Optional: true,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"enabled": {
									Type:     pluginsdk.TypeBool,
									Required: true,
								},
							},
						},
					},

					"kube_dashboard": {
						Type:     pluginsdk.TypeList,
						MaxItems: 1,
						Optional: true,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"enabled": {
									Type:     pluginsdk.TypeBool,
									Required: true,
								},
							},
						},
					},

					"http_application_routing": {
						Type:     pluginsdk.TypeList,
						MaxItems: 1,
						Optional: true,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"enabled": {
									Type:     pluginsdk.TypeBool,
									Required: true,
								},
								"http_application_routing_zone_name": {
									Type:     pluginsdk.TypeString,
									Computed: true,
								},
							},
						},
					},

					"oms_agent": {
						Type:     pluginsdk.TypeList,
						MaxItems: 1,
						Optional: true,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"enabled": {
									Type:     pluginsdk.TypeBool,
									Required: true,
								},
								"log_analytics_workspace_id": {
									Type:         pluginsdk.TypeString,
									Optional:     true,
									ValidateFunc: logAnalyticsValidate.LogAnalyticsWorkspaceID,
								},
								"oms_agent_identity": {
									Type:     pluginsdk.TypeList,
									Computed: true,
									Elem: &pluginsdk.Resource{
										Schema: map[string]*pluginsdk.Schema{
											"client_id": {
												Type:     pluginsdk.TypeString,
												Computed: true,
											},
											"object_id": {
												Type:     pluginsdk.TypeString,
												Computed: true,
											},
											"user_assigned_identity_id": {
												Type:     pluginsdk.TypeString,
												Computed: true,
											},
										},
									},
								},
							},
						},
					},

					"ingress_application_gateway": {
						Type:     pluginsdk.TypeList,
						MaxItems: 1,
						Optional: true,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"enabled": {
									Type:     pluginsdk.TypeBool,
									Required: true,
								},
								"gateway_id": {
									Type:     pluginsdk.TypeString,
									Optional: true,
									ConflictsWith: []string{
										"addon_profile.0.ingress_application_gateway.0.subnet_cidr",
										"addon_profile.0.ingress_application_gateway.0.subnet_id",
									},
									ValidateFunc: applicationGatewayValidate.ApplicationGatewayID,
								},
								"gateway_name": {
									Type:         pluginsdk.TypeString,
									Optional:     true,
									ValidateFunc: validation.StringIsNotEmpty,
								},
								"subnet_cidr": {
									Type:     pluginsdk.TypeString,
									Optional: true,
									ConflictsWith: []string{
										"addon_profile.0.ingress_application_gateway.0.gateway_id",
										"addon_profile.0.ingress_application_gateway.0.subnet_id",
									},
									ValidateFunc: commonValidate.CIDR,
								},
								"subnet_id": {
									Type:     pluginsdk.TypeString,
									Optional: true,
									ConflictsWith: []string{
										"addon_profile.0.ingress_application_gateway.0.gateway_id",
										"addon_profile.0.ingress_application_gateway.0.subnet_cidr",
									},
									ValidateFunc: subnetValidate.SubnetID,
								},
								"effective_gateway_id": {
									Type:     pluginsdk.TypeString,
									Computed: true,
								},
								"ingress_application_gateway_identity": {
									Type:     pluginsdk.TypeList,
									Computed: true,
									Elem: &pluginsdk.Resource{
										Schema: map[string]*pluginsdk.Schema{
											"client_id": {
												Type:     pluginsdk.TypeString,
												Computed: true,
											},
											"object_id": {
												Type:     pluginsdk.TypeString,
												Computed: true,
											},
											"user_assigned_identity_id": {
												Type:     pluginsdk.TypeString,
												Computed: true,
											},
										},
									},
								},
							},
						},
					},

					"open_service_mesh": {
						Type:     pluginsdk.TypeList,
						MaxItems: 1,
						Optional: true,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"enabled": {
									Type:     pluginsdk.TypeBool,
									Required: true,
								},
							},
						},
					},
					"azure_keyvault_secrets_provider": {
						Type:     pluginsdk.TypeList,
						MaxItems: 1,
						Optional: true,
						Elem: &pluginsdk.Resource{
							Schema: map[string]*pluginsdk.Schema{
								"enabled": {
									Type:     pluginsdk.TypeBool,
									Required: true,
								},
								"secret_rotation_enabled": {
									Type:     pluginsdk.TypeBool,
									Default:  false,
									Optional: true,
								},
								"secret_rotation_interval": {
									Type:         pluginsdk.TypeString,
									Optional:     true,
									Default:      "2m",
									ValidateFunc: containerValidate.Duration,
								},
								"secret_identity": {
									Type:     pluginsdk.TypeList,
									Computed: true,
									Elem: &pluginsdk.Resource{
										Schema: map[string]*pluginsdk.Schema{
											"client_id": {
												Type:     pluginsdk.TypeString,
												Computed: true,
											},
											"object_id": {
												Type:     pluginsdk.TypeString,
												Computed: true,
											},
											"user_assigned_identity_id": {
												Type:     pluginsdk.TypeString,
												Computed: true,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
	}
	return &pluginsdk.Schema{
		Type:     pluginsdk.TypeList,
		MaxItems: 1,
		Optional: true,
		Computed: true,
		Elem: &pluginsdk.Resource{
			Schema: map[string]*pluginsdk.Schema{
				"aci_connector_linux": {
					Type:     pluginsdk.TypeList,
					MaxItems: 1,
					Optional: true,
					Elem: &pluginsdk.Resource{
						Schema: map[string]*pluginsdk.Schema{
							"subnet_name": {
								Type:         pluginsdk.TypeString,
								Required:     true,
								ValidateFunc: validation.StringIsNotEmpty,
							},
						},
					},
				},

				"azure_policy_enabled": {
					Type:     pluginsdk.TypeBool,
					Optional: true,
					Default:  false,
				},

				"kube_dashboard_enabled": {
					Type:     pluginsdk.TypeBool,
					Optional: true,
					Default:  false,
				},

				"http_application_routing_enabled": {
					Type:     pluginsdk.TypeBool,
					Optional: true,
					Default:  false,
				},
				"http_application_routing_zone_name": {
					Type:     pluginsdk.TypeString,
					Computed: true,
				},

				"oms_agent": {
					Type:     pluginsdk.TypeList,
					MaxItems: 1,
					Optional: true,
					Elem: &pluginsdk.Resource{
						Schema: map[string]*pluginsdk.Schema{
							"log_analytics_workspace_id": {
								Type:         pluginsdk.TypeString,
								Required:     true,
								ValidateFunc: logAnalyticsValidate.LogAnalyticsWorkspaceID,
							},
							"oms_agent_identity": {
								Type:     pluginsdk.TypeList,
								Computed: true,
								Elem: &pluginsdk.Resource{
									Schema: map[string]*pluginsdk.Schema{
										"client_id": {
											Type:     pluginsdk.TypeString,
											Computed: true,
										},
										"object_id": {
											Type:     pluginsdk.TypeString,
											Computed: true,
										},
										"user_assigned_identity_id": {
											Type:     pluginsdk.TypeString,
											Computed: true,
										},
									},
								},
							},
						},
					},
				},

				"ingress_application_gateway": {
					Type:     pluginsdk.TypeList,
					MaxItems: 1,
					Optional: true,
					Elem: &pluginsdk.Resource{
						Schema: map[string]*pluginsdk.Schema{
							"gateway_id": {
								Type:     pluginsdk.TypeString,
								Optional: true,
								ConflictsWith: []string{
									"addon_profile.0.ingress_application_gateway.0.subnet_cidr",
									"addon_profile.0.ingress_application_gateway.0.subnet_id",
								},
								AtLeastOneOf: []string{
									"addon_profile.0.ingress_application_gateway.0.gateway_id",
									"addon_profile.0.ingress_application_gateway.0.subnet_cidr",
									"addon_profile.0.ingress_application_gateway.0.subnet_id",
								},
								ValidateFunc: applicationGatewayValidate.ApplicationGatewayID,
							},
							"gateway_name": {
								Type:         pluginsdk.TypeString,
								Optional:     true,
								ValidateFunc: validation.StringIsNotEmpty,
							},
							"subnet_cidr": {
								Type:     pluginsdk.TypeString,
								Optional: true,
								ConflictsWith: []string{
									"addon_profile.0.ingress_application_gateway.0.gateway_id",
									"addon_profile.0.ingress_application_gateway.0.subnet_id",
								},
								AtLeastOneOf: []string{
									"addon_profile.0.ingress_application_gateway.0.gateway_id",
									"addon_profile.0.ingress_application_gateway.0.subnet_cidr",
									"addon_profile.0.ingress_application_gateway.0.subnet_id",
								},
								ValidateFunc: commonValidate.CIDR,
							},
							"subnet_id": {
								Type:     pluginsdk.TypeString,
								Optional: true,
								ConflictsWith: []string{
									"addon_profile.0.ingress_application_gateway.0.gateway_id",
									"addon_profile.0.ingress_application_gateway.0.subnet_cidr",
								},
								AtLeastOneOf: []string{
									"addon_profile.0.ingress_application_gateway.0.gateway_id",
									"addon_profile.0.ingress_application_gateway.0.subnet_cidr",
									"addon_profile.0.ingress_application_gateway.0.subnet_id",
								},
								ValidateFunc: subnetValidate.SubnetID,
							},
							"effective_gateway_id": {
								Type:     pluginsdk.TypeString,
								Computed: true,
							},
							"ingress_application_gateway_identity": {
								Type:     pluginsdk.TypeList,
								Computed: true,
								Elem: &pluginsdk.Resource{
									Schema: map[string]*pluginsdk.Schema{
										"client_id": {
											Type:     pluginsdk.TypeString,
											Computed: true,
										},
										"object_id": {
											Type:     pluginsdk.TypeString,
											Computed: true,
										},
										"user_assigned_identity_id": {
											Type:     pluginsdk.TypeString,
											Computed: true,
										},
									},
								},
							},
						},
					},
				},

				"open_service_mesh_enabled": {
					Type:     pluginsdk.TypeBool,
					Optional: true,
					Default:  false,
				},

				"azure_keyvault_secrets_provider": {
					Type:     pluginsdk.TypeList,
					MaxItems: 1,
					Optional: true,
					Elem: &pluginsdk.Resource{
						Schema: map[string]*pluginsdk.Schema{
							"secret_rotation_enabled": {
								Type:     pluginsdk.TypeBool,
								Default:  false,
								Optional: true,
								AtLeastOneOf: []string{
									"addon_profile.0.azure_keyvault_secrets_provider.0.secret_rotation_enabled",
									"addon_profile.0.azure_keyvault_secrets_provider.0.secret_rotation_interval",
								},
							},
							"secret_rotation_interval": {
								Type:     pluginsdk.TypeString,
								Optional: true,
								Default:  "2m",
								AtLeastOneOf: []string{
									"addon_profile.0.azure_keyvault_secrets_provider.0.secret_rotation_enabled",
									"addon_profile.0.azure_keyvault_secrets_provider.0.secret_rotation_interval",
								},
								ValidateFunc: containerValidate.Duration,
							},
							"secret_identity": {
								Type:     pluginsdk.TypeList,
								Computed: true,
								Elem: &pluginsdk.Resource{
									Schema: map[string]*pluginsdk.Schema{
										"client_id": {
											Type:     pluginsdk.TypeString,
											Computed: true,
										},
										"object_id": {
											Type:     pluginsdk.TypeString,
											Computed: true,
										},
										"user_assigned_identity_id": {
											Type:     pluginsdk.TypeString,
											Computed: true,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func expandKubernetesAddOnProfiles(d *pluginsdk.ResourceData, input []interface{}, env azure.Environment) (*map[string]*containerservice.ManagedClusterAddonProfile, error) {
	disabled := containerservice.ManagedClusterAddonProfile{
		Enabled: utils.Bool(false),
	}

	profiles := map[string]*containerservice.ManagedClusterAddonProfile{
		aciConnectorKey:                 &disabled,
		azurePolicyKey:                  &disabled,
		kubernetesDashboardKey:          &disabled,
		httpApplicationRoutingKey:       &disabled,
		omsAgentKey:                     &disabled,
		ingressApplicationGatewayKey:    &disabled,
		openServiceMeshKey:              &disabled,
		azureKeyvaultSecretsProviderKey: &disabled,
	}

	if len(input) == 0 || input[0] == nil {
		return filterUnsupportedKubernetesAddOns(profiles, env)
	}

	profile := input[0].(map[string]interface{})
	addonProfiles := map[string]*containerservice.ManagedClusterAddonProfile{}

	if features.ThreePointOh() {
		if ok := d.HasChange("addon_profile.0.http_application_routing_enabled"); ok {
			addonProfiles[httpApplicationRoutingKey] = &containerservice.ManagedClusterAddonProfile{
				Enabled: utils.Bool(profile["http_application_routing_enabled"].(bool)),
			}
		}

		omsAgent := profile["oms_agent"].([]interface{})
		if len(omsAgent) > 0 && omsAgent[0] != nil {
			value := omsAgent[0].(map[string]interface{})
			config := make(map[string]*string)

			if workspaceID, ok := value["log_analytics_workspace_id"]; ok && workspaceID != "" {
				lawid, err := laparse.LogAnalyticsWorkspaceID(workspaceID.(string))
				if err != nil {
					return nil, fmt.Errorf("parsing Log Analytics Workspace ID: %+v", err)
				}
				config["logAnalyticsWorkspaceResourceID"] = utils.String(lawid.ID())
			}

			addonProfiles[omsAgentKey] = &containerservice.ManagedClusterAddonProfile{
				Enabled: utils.Bool(true),
				Config:  config,
			}
		}

		aciConnector := profile["aci_connector_linux"].([]interface{})
		if len(aciConnector) > 0 && aciConnector[0] != nil {
			value := aciConnector[0].(map[string]interface{})
			config := make(map[string]*string)

			if subnetName, ok := value["subnet_name"]; ok && subnetName != "" {
				config["SubnetName"] = utils.String(subnetName.(string))
			}

			addonProfiles[aciConnectorKey] = &containerservice.ManagedClusterAddonProfile{
				Enabled: utils.Bool(true),
				Config:  config,
			}
		}

		if ok := d.HasChange("addon_profile.0.kube_dashboard_enabled"); ok {
			addonProfiles[kubernetesDashboardKey] = &containerservice.ManagedClusterAddonProfile{
				Enabled: utils.Bool(profile["kube_dashboard_enabled"].(bool)),
				Config:  nil,
			}
		}

		if ok := d.HasChange("addon_profile.0.azure_policy_enabled"); ok {
			v := profile["azure_policy_enabled"].(bool)
			props := &containerservice.ManagedClusterAddonProfile{
				Enabled: utils.Bool(v),
			}
			if v != false {
				props.Config = map[string]*string{
					"version": utils.String("v2"),
				}
			}
			addonProfiles[azurePolicyKey] = props
		}

		ingressApplicationGateway := profile["ingress_application_gateway"].([]interface{})
		if len(ingressApplicationGateway) > 0 && ingressApplicationGateway[0] != nil {
			value := ingressApplicationGateway[0].(map[string]interface{})
			config := make(map[string]*string)

			if gatewayId, ok := value["gateway_id"]; ok && gatewayId != "" {
				config["applicationGatewayId"] = utils.String(gatewayId.(string))
			}

			if gatewayName, ok := value["gateway_name"]; ok && gatewayName != "" {
				config["applicationGatewayName"] = utils.String(gatewayName.(string))
			}

			if subnetCIDR, ok := value["subnet_cidr"]; ok && subnetCIDR != "" {
				config["subnetCIDR"] = utils.String(subnetCIDR.(string))
			}

			if subnetId, ok := value["subnet_id"]; ok && subnetId != "" {
				config["subnetId"] = utils.String(subnetId.(string))
			}

			addonProfiles[ingressApplicationGatewayKey] = &containerservice.ManagedClusterAddonProfile{
				Enabled: utils.Bool(true),
				Config:  config,
			}
		}

		if ok := d.HasChange("addon_profile.0.open_service_mesh_enabled"); ok {
			addonProfiles[openServiceMeshKey] = &containerservice.ManagedClusterAddonProfile{
				Enabled: utils.Bool(profile["open_service_mesh_enabled"].(bool)),
				Config:  nil,
			}
		}

		azureKeyVaultSecretsProvider := profile["azure_keyvault_secrets_provider"].([]interface{})
		if len(azureKeyVaultSecretsProvider) > 0 && azureKeyVaultSecretsProvider[0] != nil {
			value := azureKeyVaultSecretsProvider[0].(map[string]interface{})
			config := make(map[string]*string)

			enableSecretRotation := "false"
			if value["secret_rotation_enabled"].(bool) {
				enableSecretRotation = "true"
			}
			config["enableSecretRotation"] = utils.String(enableSecretRotation)
			config["rotationPollInterval"] = utils.String(value["secret_rotation_interval"].(string))

			addonProfiles[azureKeyvaultSecretsProviderKey] = &containerservice.ManagedClusterAddonProfile{
				Enabled: utils.Bool(true),
				Config:  config,
			}
		}

		return filterUnsupportedKubernetesAddOns(addonProfiles, env)
	} else {
		// TODO 3.0 - Remove this block
		httpApplicationRouting := profile["http_application_routing"].([]interface{})
		if len(httpApplicationRouting) > 0 && httpApplicationRouting[0] != nil {
			value := httpApplicationRouting[0].(map[string]interface{})
			enabled := value["enabled"].(bool)
			addonProfiles[httpApplicationRoutingKey] = &containerservice.ManagedClusterAddonProfile{
				Enabled: utils.Bool(enabled),
			}
		}

		omsAgent := profile["oms_agent"].([]interface{})
		if len(omsAgent) > 0 && omsAgent[0] != nil {
			value := omsAgent[0].(map[string]interface{})
			config := make(map[string]*string)
			enabled := value["enabled"].(bool)

			if workspaceID, ok := value["log_analytics_workspace_id"]; ok && workspaceID != "" {
				lawid, err := laparse.LogAnalyticsWorkspaceID(workspaceID.(string))
				if err != nil {
					return nil, fmt.Errorf("parsing Log Analytics Workspace ID: %+v", err)
				}
				config["logAnalyticsWorkspaceResourceID"] = utils.String(lawid.ID())
			}

			addonProfiles[omsAgentKey] = &containerservice.ManagedClusterAddonProfile{
				Enabled: utils.Bool(enabled),
				Config:  config,
			}
		}

		aciConnector := profile["aci_connector_linux"].([]interface{})
		if len(aciConnector) > 0 && aciConnector[0] != nil {
			value := aciConnector[0].(map[string]interface{})
			config := make(map[string]*string)
			enabled := value["enabled"].(bool)

			if subnetName, ok := value["subnet_name"]; ok && subnetName != "" {
				config["SubnetName"] = utils.String(subnetName.(string))
			}

			addonProfiles[aciConnectorKey] = &containerservice.ManagedClusterAddonProfile{
				Enabled: utils.Bool(enabled),
				Config:  config,
			}
		}

		kubeDashboard := profile["kube_dashboard"].([]interface{})
		if len(kubeDashboard) > 0 && kubeDashboard[0] != nil {
			value := kubeDashboard[0].(map[string]interface{})
			enabled := value["enabled"].(bool)

			addonProfiles[kubernetesDashboardKey] = &containerservice.ManagedClusterAddonProfile{
				Enabled: utils.Bool(enabled),
				Config:  nil,
			}
		}

		azurePolicy := profile["azure_policy"].([]interface{})
		if len(azurePolicy) > 0 && azurePolicy[0] != nil {
			value := azurePolicy[0].(map[string]interface{})
			enabled := value["enabled"].(bool)

			addonProfiles[azurePolicyKey] = &containerservice.ManagedClusterAddonProfile{
				Enabled: utils.Bool(enabled),
				Config: map[string]*string{
					"version": utils.String("v2"),
				},
			}
		}

		ingressApplicationGateway := profile["ingress_application_gateway"].([]interface{})
		if len(ingressApplicationGateway) > 0 && ingressApplicationGateway[0] != nil {
			value := ingressApplicationGateway[0].(map[string]interface{})
			config := make(map[string]*string)
			enabled := value["enabled"].(bool)

			if gatewayId, ok := value["gateway_id"]; ok && gatewayId != "" {
				config["applicationGatewayId"] = utils.String(gatewayId.(string))
			}

			if gatewayName, ok := value["gateway_name"]; ok && gatewayName != "" {
				config["applicationGatewayName"] = utils.String(gatewayName.(string))
			}

			if subnetCIDR, ok := value["subnet_cidr"]; ok && subnetCIDR != "" {
				config["subnetCIDR"] = utils.String(subnetCIDR.(string))
			}

			if subnetId, ok := value["subnet_id"]; ok && subnetId != "" {
				config["subnetId"] = utils.String(subnetId.(string))
			}

			addonProfiles[ingressApplicationGatewayKey] = &containerservice.ManagedClusterAddonProfile{
				Enabled: utils.Bool(enabled),
				Config:  config,
			}
		}

		openServiceMesh := profile["open_service_mesh"].([]interface{})
		if len(openServiceMesh) > 0 && openServiceMesh[0] != nil {
			value := openServiceMesh[0].(map[string]interface{})
			enabled := value["enabled"].(bool)

			addonProfiles[openServiceMeshKey] = &containerservice.ManagedClusterAddonProfile{
				Enabled: utils.Bool(enabled),
				Config:  nil,
			}
		}

		azureKeyvaultSecretsProvider := profile["azure_keyvault_secrets_provider"].([]interface{})
		if len(azureKeyvaultSecretsProvider) > 0 && azureKeyvaultSecretsProvider[0] != nil {
			value := azureKeyvaultSecretsProvider[0].(map[string]interface{})
			config := make(map[string]*string)
			enabled := value["enabled"].(bool)

			enableSecretRotation := "false"
			if value["secret_rotation_enabled"].(bool) {
				enableSecretRotation = "true"
			}
			config["enableSecretRotation"] = utils.String(enableSecretRotation)
			config["rotationPollInterval"] = utils.String(value["secret_rotation_interval"].(string))

			addonProfiles[azureKeyvaultSecretsProviderKey] = &containerservice.ManagedClusterAddonProfile{
				Enabled: utils.Bool(enabled),
				Config:  config,
			}
		}

		return filterUnsupportedKubernetesAddOns(addonProfiles, env)
	}
}

func filterUnsupportedKubernetesAddOns(input map[string]*containerservice.ManagedClusterAddonProfile, env azure.Environment) (*map[string]*containerservice.ManagedClusterAddonProfile, error) {
	filter := func(input map[string]*containerservice.ManagedClusterAddonProfile, key string) (*map[string]*containerservice.ManagedClusterAddonProfile, error) {
		output := input
		if v, ok := output[key]; ok {
			if v.Enabled != nil && *v.Enabled {
				return nil, fmt.Errorf("The addon %q is not supported for a Kubernetes Cluster located in %q", key, env.Name)
			}

			// otherwise it's disabled by default, so just remove it
			delete(output, key)
		}

		return &output, nil
	}

	output := input
	if unsupportedAddons, ok := unsupportedAddonsForEnvironment[env.Name]; ok {
		for _, key := range unsupportedAddons {
			out, err := filter(output, key)
			if err != nil {
				return nil, err
			}

			output = *out
		}
	}
	return &output, nil
}

func flattenKubernetesAddOnProfiles(profile map[string]*containerservice.ManagedClusterAddonProfile) []interface{} {
	if features.ThreePointOh() {
		aciConnectors := make([]interface{}, 0)
		if aciConnector := kubernetesAddonProfileLocate(profile, aciConnectorKey); aciConnector != nil {
			subnetName := ""
			if v := aciConnector.Config["SubnetName"]; v != nil {
				subnetName = *v
			}

			aciConnectors = append(aciConnectors, map[string]interface{}{
				"subnet_name": subnetName,
			})
		}

		azurePolicyEnabled := false
		if azurePolicy := kubernetesAddonProfileLocate(profile, azurePolicyKey); azurePolicy != nil {
			if enabledVal := azurePolicy.Enabled; enabledVal != nil {
				azurePolicyEnabled = *enabledVal
			}
		}

		httpApplicationRoutingEnabled := false
		httpApplicationRoutingZone := ""
		if httpApplicationRouting := kubernetesAddonProfileLocate(profile, httpApplicationRoutingKey); httpApplicationRouting != nil {
			if enabledVal := httpApplicationRouting.Enabled; enabledVal != nil {
				httpApplicationRoutingEnabled = *enabledVal
			}

			if v := kubernetesAddonProfilelocateInConfig(httpApplicationRouting.Config, "HTTPApplicationRoutingZoneName"); v != nil {
				httpApplicationRoutingZone = *v
			}
		}

		kubeDashboardEnabled := false
		if kubeDashboard := kubernetesAddonProfileLocate(profile, kubernetesDashboardKey); kubeDashboard != nil {
			if enabledVal := kubeDashboard.Enabled; enabledVal != nil {
				kubeDashboardEnabled = *enabledVal
			}
		}

		omsAgents := make([]interface{}, 0)
		if omsAgent := kubernetesAddonProfileLocate(profile, omsAgentKey); omsAgent != nil {
			workspaceID := ""
			if v := kubernetesAddonProfilelocateInConfig(omsAgent.Config, "logAnalyticsWorkspaceResourceID"); v != nil {
				if lawid, err := laparse.LogAnalyticsWorkspaceID(*v); err == nil {
					workspaceID = lawid.ID()
				}
			}

			omsAgentIdentity := flattenKubernetesClusterAddOnIdentityProfile(omsAgent.Identity)

			omsAgents = append(omsAgents, map[string]interface{}{
				"log_analytics_workspace_id": workspaceID,
				"oms_agent_identity":         omsAgentIdentity,
			})
		}

		ingressApplicationGateways := make([]interface{}, 0)
		if ingressApplicationGateway := kubernetesAddonProfileLocate(profile, ingressApplicationGatewayKey); ingressApplicationGateway != nil {
			gatewayId := ""
			if v := kubernetesAddonProfilelocateInConfig(ingressApplicationGateway.Config, "applicationGatewayId"); v != nil {
				gatewayId = *v
			}

			gatewayName := ""
			if v := kubernetesAddonProfilelocateInConfig(ingressApplicationGateway.Config, "applicationGatewayName"); v != nil {
				gatewayName = *v
			}

			effectiveGatewayId := ""
			if v := kubernetesAddonProfilelocateInConfig(ingressApplicationGateway.Config, "effectiveApplicationGatewayId"); v != nil {
				effectiveGatewayId = *v
			}

			subnetCIDR := ""
			if v := kubernetesAddonProfilelocateInConfig(ingressApplicationGateway.Config, "subnetCIDR"); v != nil {
				subnetCIDR = *v
			}

			subnetId := ""
			if v := kubernetesAddonProfilelocateInConfig(ingressApplicationGateway.Config, "subnetId"); v != nil {
				subnetId = *v
			}

			ingressApplicationGatewayIdentity := flattenKubernetesClusterAddOnIdentityProfile(ingressApplicationGateway.Identity)

			ingressApplicationGateways = append(ingressApplicationGateways, map[string]interface{}{
				"gateway_id":                           gatewayId,
				"gateway_name":                         gatewayName,
				"effective_gateway_id":                 effectiveGatewayId,
				"subnet_cidr":                          subnetCIDR,
				"subnet_id":                            subnetId,
				"ingress_application_gateway_identity": ingressApplicationGatewayIdentity,
			})
		}

		openServiceMeshEnabled := false
		if openServiceMesh := kubernetesAddonProfileLocate(profile, openServiceMeshKey); openServiceMesh != nil {
			if enabledVal := openServiceMesh.Enabled; enabledVal != nil {
				openServiceMeshEnabled = *enabledVal
			}
		}

		azureKeyVaultSecretsProviders := make([]interface{}, 0)
		if azureKeyVaultSecretsProvider := kubernetesAddonProfileLocate(profile, azureKeyvaultSecretsProviderKey); azureKeyVaultSecretsProvider != nil {
			enableSecretRotation := false
			if v := kubernetesAddonProfilelocateInConfig(azureKeyVaultSecretsProvider.Config, "enableSecretRotation"); v != nil && *v != "false" {
				enableSecretRotation = true
			}

			rotationPollInterval := ""
			if v := kubernetesAddonProfilelocateInConfig(azureKeyVaultSecretsProvider.Config, "rotationPollInterval"); v != nil {
				rotationPollInterval = *v
			}

			azureKeyvaultSecretsProviderIdentity := flattenKubernetesClusterAddOnIdentityProfile(azureKeyVaultSecretsProvider.Identity)

			azureKeyVaultSecretsProviders = append(azureKeyVaultSecretsProviders, map[string]interface{}{
				"secret_rotation_enabled":  enableSecretRotation,
				"secret_rotation_interval": rotationPollInterval,
				"secret_identity":          azureKeyvaultSecretsProviderIdentity,
			})
		}

		return []interface{}{
			map[string]interface{}{
				"aci_connector_linux":                aciConnectors,
				"azure_policy_enabled":               azurePolicyEnabled,
				"http_application_routing_enabled":   httpApplicationRoutingEnabled,
				"http_application_routing_zone_name": httpApplicationRoutingZone,
				"kube_dashboard_enabled":             kubeDashboardEnabled,
				"oms_agent":                          omsAgents,
				"ingress_application_gateway":        ingressApplicationGateways,
				"open_service_mesh_enabled":          openServiceMeshEnabled,
				"azure_keyvault_secrets_provider":    azureKeyVaultSecretsProviders,
			},
		}
	} else {
		// TODO 3.0 - Remove this block
		aciConnectors := make([]interface{}, 0)
		if aciConnector := kubernetesAddonProfileLocate(profile, aciConnectorKey); aciConnector != nil {
			enabled := false
			if enabledVal := aciConnector.Enabled; enabledVal != nil {
				enabled = *enabledVal
			}

			subnetName := ""
			if v := aciConnector.Config["SubnetName"]; v != nil {
				subnetName = *v
			}

			aciConnectors = append(aciConnectors, map[string]interface{}{
				"enabled":     enabled,
				"subnet_name": subnetName,
			})
		}

		azurePolicies := make([]interface{}, 0)
		if azurePolicy := kubernetesAddonProfileLocate(profile, azurePolicyKey); azurePolicy != nil {
			enabled := false
			if enabledVal := azurePolicy.Enabled; enabledVal != nil {
				enabled = *enabledVal
			}

			azurePolicies = append(azurePolicies, map[string]interface{}{
				"enabled": enabled,
			})
		}

		httpApplicationRoutes := make([]interface{}, 0)
		if httpApplicationRouting := kubernetesAddonProfileLocate(profile, httpApplicationRoutingKey); httpApplicationRouting != nil {
			enabled := false
			if enabledVal := httpApplicationRouting.Enabled; enabledVal != nil {
				enabled = *enabledVal
			}

			zoneName := ""
			if v := kubernetesAddonProfilelocateInConfig(httpApplicationRouting.Config, "HTTPApplicationRoutingZoneName"); v != nil {
				zoneName = *v
			}

			httpApplicationRoutes = append(httpApplicationRoutes, map[string]interface{}{
				"enabled":                            enabled,
				"http_application_routing_zone_name": zoneName,
			})
		}

		kubeDashboards := make([]interface{}, 0)
		if kubeDashboard := kubernetesAddonProfileLocate(profile, kubernetesDashboardKey); kubeDashboard != nil {
			enabled := false
			if enabledVal := kubeDashboard.Enabled; enabledVal != nil {
				enabled = *enabledVal
			}

			kubeDashboards = append(kubeDashboards, map[string]interface{}{
				"enabled": enabled,
			})
		}

		omsAgents := make([]interface{}, 0)
		if omsAgent := kubernetesAddonProfileLocate(profile, omsAgentKey); omsAgent != nil {
			enabled := false
			if enabledVal := omsAgent.Enabled; enabledVal != nil {
				enabled = *enabledVal
			}

			workspaceID := ""
			if v := kubernetesAddonProfilelocateInConfig(omsAgent.Config, "logAnalyticsWorkspaceResourceID"); v != nil {
				if lawid, err := laparse.LogAnalyticsWorkspaceID(*v); err == nil {
					workspaceID = lawid.ID()
				}
			}

			omsagentIdentity := flattenKubernetesClusterAddOnIdentityProfile(omsAgent.Identity)

			omsAgents = append(omsAgents, map[string]interface{}{
				"enabled":                    enabled,
				"log_analytics_workspace_id": workspaceID,
				"oms_agent_identity":         omsagentIdentity,
			})
		}

		ingressApplicationGateways := make([]interface{}, 0)
		if ingressApplicationGateway := kubernetesAddonProfileLocate(profile, ingressApplicationGatewayKey); ingressApplicationGateway != nil {
			enabled := false
			if enabledVal := ingressApplicationGateway.Enabled; enabledVal != nil {
				enabled = *enabledVal
			}

			gatewayId := ""
			if v := kubernetesAddonProfilelocateInConfig(ingressApplicationGateway.Config, "applicationGatewayId"); v != nil {
				gatewayId = *v
			}

			gatewayName := ""
			if v := kubernetesAddonProfilelocateInConfig(ingressApplicationGateway.Config, "applicationGatewayName"); v != nil {
				gatewayName = *v
			}

			effectiveGatewayId := ""
			if v := kubernetesAddonProfilelocateInConfig(ingressApplicationGateway.Config, "effectiveApplicationGatewayId"); v != nil {
				effectiveGatewayId = *v
			}

			subnetCIDR := ""
			if v := kubernetesAddonProfilelocateInConfig(ingressApplicationGateway.Config, "subnetCIDR"); v != nil {
				subnetCIDR = *v
			}

			subnetId := ""
			if v := kubernetesAddonProfilelocateInConfig(ingressApplicationGateway.Config, "subnetId"); v != nil {
				subnetId = *v
			}

			ingressApplicationGatewayIdentity := flattenKubernetesClusterAddOnIdentityProfile(ingressApplicationGateway.Identity)

			ingressApplicationGateways = append(ingressApplicationGateways, map[string]interface{}{
				"enabled":                              enabled,
				"gateway_id":                           gatewayId,
				"gateway_name":                         gatewayName,
				"effective_gateway_id":                 effectiveGatewayId,
				"subnet_cidr":                          subnetCIDR,
				"subnet_id":                            subnetId,
				"ingress_application_gateway_identity": ingressApplicationGatewayIdentity,
			})
		}

		openServiceMeshes := make([]interface{}, 0)
		if openServiceMesh := kubernetesAddonProfileLocate(profile, openServiceMeshKey); openServiceMesh != nil {
			enabled := false
			if enabledVal := openServiceMesh.Enabled; enabledVal != nil {
				enabled = *enabledVal
			}

			openServiceMeshes = append(openServiceMeshes, map[string]interface{}{
				"enabled": enabled,
			})
		}

		azureKeyvaultSecretsProviders := make([]interface{}, 0)
		if azureKeyvaultSecretsProvider := kubernetesAddonProfileLocate(profile, azureKeyvaultSecretsProviderKey); azureKeyvaultSecretsProvider != nil {
			enabled := false
			if enabledVal := azureKeyvaultSecretsProvider.Enabled; enabledVal != nil {
				enabled = *enabledVal
			}
			enableSecretRotation := false
			if v := kubernetesAddonProfilelocateInConfig(azureKeyvaultSecretsProvider.Config, "enableSecretRotation"); v != nil && *v != "false" {
				enableSecretRotation = true
			}
			rotationPollInterval := ""
			if v := kubernetesAddonProfilelocateInConfig(azureKeyvaultSecretsProvider.Config, "rotationPollInterval"); v != nil {
				rotationPollInterval = *v
			}

			azureKeyvaultSecretsProviderIdentity := flattenKubernetesClusterAddOnIdentityProfile(azureKeyvaultSecretsProvider.Identity)

			azureKeyvaultSecretsProviders = append(azureKeyvaultSecretsProviders, map[string]interface{}{
				"enabled":                  enabled,
				"secret_rotation_enabled":  enableSecretRotation,
				"secret_rotation_interval": rotationPollInterval,
				"secret_identity":          azureKeyvaultSecretsProviderIdentity,
			})
		}

		// this is a UX hack, since if the top level block isn't defined everything should be turned off
		if len(aciConnectors) == 0 && len(azurePolicies) == 0 && len(httpApplicationRoutes) == 0 && len(kubeDashboards) == 0 && len(omsAgents) == 0 && len(ingressApplicationGateways) == 0 && len(openServiceMeshes) == 0 && len(azureKeyvaultSecretsProviders) == 0 {
			return []interface{}{}
		}

		return []interface{}{
			map[string]interface{}{
				"aci_connector_linux":             aciConnectors,
				"azure_policy":                    azurePolicies,
				"http_application_routing":        httpApplicationRoutes,
				"kube_dashboard":                  kubeDashboards,
				"oms_agent":                       omsAgents,
				"ingress_application_gateway":     ingressApplicationGateways,
				"open_service_mesh":               openServiceMeshes,
				"azure_keyvault_secrets_provider": azureKeyvaultSecretsProviders,
			},
		}
	}
}

func flattenKubernetesClusterAddOnIdentityProfile(profile *containerservice.ManagedClusterAddonProfileIdentity) []interface{} {
	if profile == nil {
		return []interface{}{}
	}

	identity := make([]interface{}, 0)
	clientID := ""
	if clientid := profile.ClientID; clientid != nil {
		clientID = *clientid
	}

	objectID := ""
	if objectid := profile.ObjectID; objectid != nil {
		objectID = *objectid
	}

	userAssignedIdentityID := ""
	if resourceid := profile.ResourceID; resourceid != nil {
		userAssignedIdentityID = *resourceid
	}

	identity = append(identity, map[string]interface{}{
		"client_id":                 clientID,
		"object_id":                 objectID,
		"user_assigned_identity_id": userAssignedIdentityID,
	})

	return identity
}

// when the Kubernetes Cluster is updated in the Portal - Azure updates the casing on the keys
// meaning what's submitted could be different to what's returned..
func kubernetesAddonProfileLocate(profile map[string]*containerservice.ManagedClusterAddonProfile, key string) *containerservice.ManagedClusterAddonProfile {
	for k, v := range profile {
		if strings.EqualFold(k, key) {
			return v
		}
	}

	return nil
}

// when the Kubernetes Cluster is updated in the Portal - Azure updates the casing on the keys
// meaning what's submitted could be different to what's returned..
// Related issue: https://github.com/Azure/azure-rest-api-specs/issues/10716
func kubernetesAddonProfilelocateInConfig(config map[string]*string, key string) *string {
	for k, v := range config {
		if strings.EqualFold(k, key) {
			return v
		}
	}

	return nil
}
