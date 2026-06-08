package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &HostInterfaceDataSource{}

func NewHostInterfaceDataSource() datasource.DataSource {
	return &HostInterfaceDataSource{}
}

type HostInterfaceDataSource struct {
	client client.Client
}

type HostInterfaceDataSourceModel struct {
	ID     types.String `tfsdk:"id"`
	HostID types.String `tfsdk:"host_id"`
	Type   types.String `tfsdk:"type"`
	UseIP  types.Bool   `tfsdk:"use_ip"`
	IP     types.String `tfsdk:"ip"`
	DNS    types.String `tfsdk:"dns"`
	Port   types.String `tfsdk:"port"`
	Main   types.Bool   `tfsdk:"main"`
	SNMP   types.Object `tfsdk:"snmp"`
}

func (d *HostInterfaceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host_interface"
}

func (d *HostInterfaceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a Zabbix host interface by ID, or by `host_id` + `type`. Exactly one lookup strategy must be used.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Unique identifier of the host interface. One of `id` or (`host_id` + `type`) must be set.",
			},
			"host_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "ID of the host. Used with `type` for composite lookup.",
			},
			"type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Interface type. One of: `agent`, `snmp`, `ipmi`, `jmx`. Used with `host_id` for composite lookup.",
				Validators: []validator.String{
					stringvalidator.OneOf("agent", "snmp", "ipmi", "jmx"),
				},
			},
			"use_ip": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the IP address (`true`) or DNS name (`false`) is used for monitoring.",
			},
			"ip": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "IP address of the interface.",
			},
			"dns": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "DNS name of the interface.",
			},
			"port": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Port used by the interface.",
			},
			"main": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this is the default interface of its type for the host.",
			},
			"snmp": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "SNMP-specific settings. Only populated for `type = \"snmp\"` interfaces.",
				Attributes: map[string]schema.Attribute{
					"version":         schema.StringAttribute{Computed: true, MarkdownDescription: "SNMP version."},
					"community":       schema.StringAttribute{Computed: true, MarkdownDescription: "SNMP community string."},
					"bulk_requests":   schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether bulk SNMP requests are used."},
					"security_name":   schema.StringAttribute{Computed: true, MarkdownDescription: "SNMPv3 security name."},
					"security_level":  schema.StringAttribute{Computed: true, MarkdownDescription: "SNMPv3 security level."},
					"auth_protocol":   schema.StringAttribute{Computed: true, MarkdownDescription: "SNMPv3 authentication protocol."},
					"auth_passphrase": schema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "SNMPv3 authentication passphrase."},
					"priv_protocol":   schema.StringAttribute{Computed: true, MarkdownDescription: "SNMPv3 privacy protocol."},
					"priv_passphrase": schema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "SNMPv3 privacy passphrase."},
					"context_name":    schema.StringAttribute{Computed: true, MarkdownDescription: "SNMPv3 context name."},
				},
			},
		},
	}
}

func (d *HostInterfaceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *HostInterfaceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data HostInterfaceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() && (data.HostID.IsNull() || data.Type.IsNull()) {
		resp.Diagnostics.AddError(
			"Missing lookup key",
			"Provide either `id`, or both `host_id` and `type`.",
		)
		return
	}

	var iface *client.HostInterface

	if !data.ID.IsNull() {
		found, err := client.HostInterfaceGet(ctx, d.client, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading host interface", err.Error())
			return
		}
		if found == nil {
			resp.Diagnostics.AddError(
				"Host interface not found",
				fmt.Sprintf("No host interface found with id %q.", data.ID.ValueString()),
			)
			return
		}
		iface = found
	} else {
		ifaceType, ok := hostInterfaceTypeMap[data.Type.ValueString()]
		if !ok {
			resp.Diagnostics.AddError("Invalid interface type", fmt.Sprintf("Unknown type %q.", data.Type.ValueString()))
			return
		}
		ifaces, err := client.HostInterfaceGetByHostAndType(ctx, d.client, data.HostID.ValueString(), ifaceType)
		if err != nil {
			resp.Diagnostics.AddError("Error reading host interface", err.Error())
			return
		}
		switch len(ifaces) {
		case 0:
			resp.Diagnostics.AddError(
				"Host interface not found",
				fmt.Sprintf("No %q interface found for host %q.", data.Type.ValueString(), data.HostID.ValueString()),
			)
			return
		case 1:
			iface = &ifaces[0]
		default:
			resp.Diagnostics.AddError(
				"Multiple host interfaces found",
				fmt.Sprintf("Found %d %q interfaces for host %q; use `id` to disambiguate.", len(ifaces), data.Type.ValueString(), data.HostID.ValueString()),
			)
			return
		}
	}

	data.ID = types.StringValue(iface.InterfaceID)
	dsModel := &HostInterfaceResourceModel{
		ID:   data.ID,
		SNMP: data.SNMP,
	}
	resp.Diagnostics.Append(clientHostInterfaceToModel(ctx, *iface, dsModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.HostID = dsModel.HostID
	data.Type = dsModel.Type
	data.UseIP = dsModel.UseIP
	data.IP = dsModel.IP
	data.DNS = dsModel.DNS
	data.Port = dsModel.Port
	data.Main = dsModel.Main
	data.SNMP = dsModel.SNMP

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
