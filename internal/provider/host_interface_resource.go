package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ resource.Resource = &HostInterfaceResource{}
var _ resource.ResourceWithImportState = &HostInterfaceResource{}
var _ resource.ResourceWithConfigValidators = &HostInterfaceResource{}

var (
	hostInterfaceTypeMap = map[string]int64{
		"agent": 1, "snmp": 2, "ipmi": 3, "jmx": 4,
	}
	hostInterfaceTypeReverseMap = map[int64]string{
		1: "agent", 2: "snmp", 3: "ipmi", 4: "jmx",
	}
	snmpVersionMap = map[string]int64{
		"v1": 1, "v2c": 2, "v3": 3,
	}
	snmpVersionReverseMap = map[int64]string{
		1: "v1", 2: "v2c", 3: "v3",
	}
	snmpSecurityLevelMap = map[string]int64{
		"noAuthNoPriv": 0, "authNoPriv": 1, "authPriv": 2,
	}
	snmpSecurityLevelReverseMap = map[int64]string{
		0: "noAuthNoPriv", 1: "authNoPriv", 2: "authPriv",
	}
	snmpAuthProtocolMap = map[string]int64{
		"md5": 0, "sha1": 1, "sha224": 2, "sha256": 3, "sha384": 4, "sha512": 5,
	}
	snmpAuthProtocolReverseMap = map[int64]string{
		0: "md5", 1: "sha1", 2: "sha224", 3: "sha256", 4: "sha384", 5: "sha512",
	}
	snmpPrivProtocolMap = map[string]int64{
		"des": 0, "aes128": 1, "aes192": 2, "aes256": 3, "aes192c": 4, "aes256c": 5,
	}
	snmpPrivProtocolReverseMap = map[int64]string{
		0: "des", 1: "aes128", 2: "aes192", 3: "aes256", 4: "aes192c", 5: "aes256c",
	}
)

var snmpAttrTypes = map[string]attr.Type{
	"version":         types.StringType,
	"community":       types.StringType,
	"bulk_requests":   types.BoolType,
	"security_name":   types.StringType,
	"security_level":  types.StringType,
	"auth_protocol":   types.StringType,
	"auth_passphrase": types.StringType,
	"priv_protocol":   types.StringType,
	"priv_passphrase": types.StringType,
	"context_name":    types.StringType,
}

func NewHostInterfaceResource() resource.Resource {
	return &HostInterfaceResource{}
}

type HostInterfaceResource struct {
	client client.Client
}

type HostInterfaceResourceModel struct {
	ID     types.String `tfsdk:"id"`
	HostID types.String `tfsdk:"host_id"`
	Type   types.String `tfsdk:"type"`
	UseIP  types.Bool   `tfsdk:"use_ip"`
	IP     types.String `tfsdk:"ip"`
	DNS    types.String `tfsdk:"dns"`
	Port   types.String `tfsdk:"port"`
	Main   types.Bool   `tfsdk:"main"`
	SNMP   types.Object `tfsdk:"snmp"` // null when not set; HostInterfaceSNMPModel attributes when set
}

type HostInterfaceSNMPModel struct {
	Version        types.String `tfsdk:"version"`
	Community      types.String `tfsdk:"community"`
	BulkRequests   types.Bool   `tfsdk:"bulk_requests"`
	SecurityName   types.String `tfsdk:"security_name"`
	SecurityLevel  types.String `tfsdk:"security_level"`
	AuthProtocol   types.String `tfsdk:"auth_protocol"`
	AuthPassphrase types.String `tfsdk:"auth_passphrase"`
	PrivProtocol   types.String `tfsdk:"priv_protocol"`
	PrivPassphrase types.String `tfsdk:"priv_passphrase"`
	ContextName    types.String `tfsdk:"context_name"`
}

func (r *HostInterfaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host_interface"
}

func (r *HostInterfaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Zabbix host interface.\n\n" +
			"~> **Note:** Multiple interfaces of the same type with `main = true` on the same host will cause a Zabbix API error at apply time. " +
			"This constraint cannot be validated at plan time across multiple resources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the host interface.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"host_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the host this interface belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Interface type. One of: `agent`, `snmp`, `ipmi`, `jmx`. Changing this forces a new resource.",
				Validators: []validator.String{
					stringvalidator.OneOf("agent", "snmp", "ipmi", "jmx"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"use_ip": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether to use the IP address (`true`) or DNS name (`false`) for monitoring. Defaults to `true`.",
			},
			"ip": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "IP address of the interface. Required when `use_ip = true`.",
			},
			"dns": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "DNS name of the interface. Required when `use_ip = false`.",
			},
			"port": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Port used by the interface.",
			},
			"main": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether this is the default interface of its type for the host. Defaults to `false`.",
			},
			"snmp": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "SNMP-specific settings. Required when `type = \"snmp\"`, must not be set for other types.",
				Attributes: map[string]schema.Attribute{
					"version": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "SNMP version. One of: `v1`, `v2c`, `v3`.",
						Validators: []validator.String{
							stringvalidator.OneOf("v1", "v2c", "v3"),
						},
					},
					"community": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
						MarkdownDescription: "SNMP community string. Used for SNMPv1 and SNMPv2c.",
					},
					"bulk_requests": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
						MarkdownDescription: "Whether to use bulk SNMP requests. Defaults to `true`.",
					},
					"security_name": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
						MarkdownDescription: "SNMPv3 security name.",
					},
					"security_level": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("noAuthNoPriv"),
						MarkdownDescription: "SNMPv3 security level. One of: `noAuthNoPriv`, `authNoPriv`, `authPriv`. Defaults to `noAuthNoPriv`.",
						Validators: []validator.String{
							stringvalidator.OneOf("noAuthNoPriv", "authNoPriv", "authPriv"),
						},
					},
					"auth_protocol": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("md5"),
						MarkdownDescription: "SNMPv3 authentication protocol. One of: `md5`, `sha1`, `sha224`, `sha256`, `sha384`, `sha512`. Defaults to `md5`.",
						Validators: []validator.String{
							stringvalidator.OneOf("md5", "sha1", "sha224", "sha256", "sha384", "sha512"),
						},
					},
					"auth_passphrase": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Default:             stringdefault.StaticString(""),
						MarkdownDescription: "SNMPv3 authentication passphrase.",
					},
					"priv_protocol": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("des"),
						MarkdownDescription: "SNMPv3 privacy protocol. One of: `des`, `aes128`, `aes192`, `aes256`, `aes192c`, `aes256c`. Defaults to `des`.",
						Validators: []validator.String{
							stringvalidator.OneOf("des", "aes128", "aes192", "aes256", "aes192c", "aes256c"),
						},
					},
					"priv_passphrase": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Default:             stringdefault.StaticString(""),
						MarkdownDescription: "SNMPv3 privacy passphrase.",
					},
					"context_name": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
						MarkdownDescription: "SNMPv3 context name.",
					},
				},
			},
		},
	}
}

func (r *HostInterfaceResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		snmpTypeConfigValidator{},
	}
}

// snmpTypeConfigValidator rejects configurations that set the snmp block when type != "snmp".
type snmpTypeConfigValidator struct{}

func (v snmpTypeConfigValidator) Description(_ context.Context) string {
	return "The snmp block can only be set when type = \"snmp\"."
}

func (v snmpTypeConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v snmpTypeConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data HostInterfaceResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Type.IsUnknown() || data.SNMP.IsUnknown() {
		return
	}

	if data.Type.ValueString() != "snmp" && !data.SNMP.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("snmp"),
			"Invalid snmp block",
			"The snmp block can only be set when type = \"snmp\".",
		)
	}
}

func (r *HostInterfaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *HostInterfaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data HostInterfaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	iface, diags := modelToClientHostInterface(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := client.HostInterfaceCreate(ctx, r.client, iface)
	if err != nil {
		resp.Diagnostics.AddError("Error creating host interface", err.Error())
		return
	}
	data.ID = types.StringValue(id)

	created, err := client.HostInterfaceGet(ctx, r.client, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading host interface after create", err.Error())
		return
	}
	if created != nil {
		resp.Diagnostics.Append(clientHostInterfaceToModel(ctx, *created, &data)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HostInterfaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data HostInterfaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	iface, err := client.HostInterfaceGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading host interface", err.Error())
		return
	}
	if iface == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(clientHostInterfaceToModel(ctx, *iface, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HostInterfaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data HostInterfaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the ID from state during update
	var state HostInterfaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ID = state.ID

	iface, diags := modelToClientHostInterface(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.HostInterfaceUpdate(ctx, r.client, iface); err != nil {
		resp.Diagnostics.AddError("Error updating host interface", err.Error())
		return
	}

	updated, err := client.HostInterfaceGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading host interface after update", err.Error())
		return
	}
	if updated != nil {
		resp.Diagnostics.Append(clientHostInterfaceToModel(ctx, *updated, &data)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HostInterfaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data HostInterfaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.HostInterfaceDelete(ctx, r.client, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting host interface", err.Error())
		return
	}
}

func (r *HostInterfaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// modelToClientHostInterface converts a HostInterfaceResourceModel to a client.HostInterface.
func modelToClientHostInterface(ctx context.Context, data HostInterfaceResourceModel) (client.HostInterface, diag.Diagnostics) {
	var diags diag.Diagnostics

	useIP := int64(0)
	if data.UseIP.ValueBool() {
		useIP = 1
	}
	main := int64(0)
	if data.Main.ValueBool() {
		main = 1
	}

	iface := client.HostInterface{
		InterfaceID: data.ID.ValueString(),
		HostID:      data.HostID.ValueString(),
		Type:        hostInterfaceTypeMap[data.Type.ValueString()],
		UseIP:       useIP,
		IP:          data.IP.ValueString(),
		DNS:         data.DNS.ValueString(),
		Port:        data.Port.ValueString(),
		Main:        main,
	}

	if !data.SNMP.IsNull() && !data.SNMP.IsUnknown() {
		var snmpModel HostInterfaceSNMPModel
		diags.Append(data.SNMP.As(ctx, &snmpModel, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return iface, diags
		}

		bulk := int64(0)
		if snmpModel.BulkRequests.ValueBool() {
			bulk = 1
		}

		iface.Details = &client.HostInterfaceSNMPDetails{
			Version:        snmpVersionMap[snmpModel.Version.ValueString()],
			Community:      snmpModel.Community.ValueString(),
			BulkRequests:   bulk,
			SecurityName:   snmpModel.SecurityName.ValueString(),
			SecurityLevel:  snmpSecurityLevelMap[snmpModel.SecurityLevel.ValueString()],
			AuthProtocol:   snmpAuthProtocolMap[snmpModel.AuthProtocol.ValueString()],
			AuthPassphrase: snmpModel.AuthPassphrase.ValueString(),
			PrivProtocol:   snmpPrivProtocolMap[snmpModel.PrivProtocol.ValueString()],
			PrivPassphrase: snmpModel.PrivPassphrase.ValueString(),
			ContextName:    snmpModel.ContextName.ValueString(),
		}
	}

	return iface, diags
}

// clientHostInterfaceToModel updates a HostInterfaceResourceModel from a client.HostInterface.
func clientHostInterfaceToModel(ctx context.Context, iface client.HostInterface, data *HostInterfaceResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.HostID = types.StringValue(iface.HostID)
	ifaceType, ok := hostInterfaceTypeReverseMap[iface.Type]
	if !ok {
		diags.AddError("Unknown interface type", fmt.Sprintf("Unrecognized interface type %d from API.", iface.Type))
		return diags
	}
	data.Type = types.StringValue(ifaceType)
	data.UseIP = types.BoolValue(iface.UseIP == 1)
	data.IP = types.StringValue(iface.IP)
	data.DNS = types.StringValue(iface.DNS)
	data.Port = types.StringValue(iface.Port)
	data.Main = types.BoolValue(iface.Main == 1)

	if iface.Details != nil {
		version, ok := snmpVersionReverseMap[iface.Details.Version]
		if !ok {
			diags.AddError("Unknown SNMP version", fmt.Sprintf("Unrecognized SNMP version %d from API.", iface.Details.Version))
			return diags
		}
		secLevel, ok := snmpSecurityLevelReverseMap[iface.Details.SecurityLevel]
		if !ok {
			diags.AddError("Unknown SNMP security level", fmt.Sprintf("Unrecognized security level %d from API.", iface.Details.SecurityLevel))
			return diags
		}
		authProto, ok := snmpAuthProtocolReverseMap[iface.Details.AuthProtocol]
		if !ok {
			diags.AddError("Unknown SNMP auth protocol", fmt.Sprintf("Unrecognized auth protocol %d from API.", iface.Details.AuthProtocol))
			return diags
		}
		privProto, ok := snmpPrivProtocolReverseMap[iface.Details.PrivProtocol]
		if !ok {
			diags.AddError("Unknown SNMP priv protocol", fmt.Sprintf("Unrecognized priv protocol %d from API.", iface.Details.PrivProtocol))
			return diags
		}

		// Preserve sensitive fields from state if API returns empty (Zabbix masks secrets on read)
		authPassphrase := types.StringValue(iface.Details.AuthPassphrase)
		privPassphrase := types.StringValue(iface.Details.PrivPassphrase)
		if !data.SNMP.IsNull() && !data.SNMP.IsUnknown() {
			var existingSNMP HostInterfaceSNMPModel
			if d := data.SNMP.As(ctx, &existingSNMP, basetypes.ObjectAsOptions{}); !d.HasError() {
				if iface.Details.AuthPassphrase == "" && !existingSNMP.AuthPassphrase.IsNull() {
					authPassphrase = existingSNMP.AuthPassphrase
				}
				if iface.Details.PrivPassphrase == "" && !existingSNMP.PrivPassphrase.IsNull() {
					privPassphrase = existingSNMP.PrivPassphrase
				}
			}
		}

		snmpObj, d := types.ObjectValue(snmpAttrTypes, map[string]attr.Value{
			"version":         types.StringValue(version),
			"community":       types.StringValue(iface.Details.Community),
			"bulk_requests":   types.BoolValue(iface.Details.BulkRequests == 1),
			"security_name":   types.StringValue(iface.Details.SecurityName),
			"security_level":  types.StringValue(secLevel),
			"auth_protocol":   types.StringValue(authProto),
			"auth_passphrase": authPassphrase,
			"priv_protocol":   types.StringValue(privProto),
			"priv_passphrase": privPassphrase,
			"context_name":    types.StringValue(iface.Details.ContextName),
		})
		diags.Append(d...)
		if !d.HasError() {
			data.SNMP = snmpObj
		}
	} else {
		data.SNMP = types.ObjectNull(snmpAttrTypes)
	}

	return diags
}
