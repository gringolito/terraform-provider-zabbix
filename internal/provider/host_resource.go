package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &HostResource{}
var _ resource.ResourceWithImportState = &HostResource{}

var (
	hostStatusMap = map[string]int64{
		"enabled": 0, "disabled": 1,
	}
	hostStatusReverseMap = map[int64]string{
		0: "enabled", 1: "disabled",
	}
	hostInventoryModeMap = map[string]int64{
		"disabled": -1, "manual": 0, "automatic": 1,
	}
	hostInventoryModeReverseMap = map[int64]string{
		-1: "disabled", 0: "manual", 1: "automatic",
	}

	hostTagAttrTypes = map[string]attr.Type{
		"name":  types.StringType,
		"value": types.StringType,
	}
)

func NewHostResource() resource.Resource {
	return &HostResource{}
}

type HostResource struct {
	client client.Client
}

type HostResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Host          types.String `tfsdk:"host"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	Status        types.String `tfsdk:"status"`
	HostGroupIDs  types.Set    `tfsdk:"host_group_ids"`
	Tags          types.Set    `tfsdk:"tags"`
	InventoryMode types.String `tfsdk:"inventory_mode"`
	ProxyID       types.String `tfsdk:"proxy_id"`
}

type HostTagModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

func (r *HostResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host"
}

func (r *HostResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Zabbix host.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the host.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"host": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Technical name of the host. Must be unique within Zabbix.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Visible display name of the host. Defaults to the technical name if not set.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Description of the host.",
			},
			"status": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("enabled"),
				MarkdownDescription: "Monitoring status of the host. One of: `enabled`, `disabled`. Defaults to `enabled`.",
				Validators: []validator.String{
					stringvalidator.OneOf("enabled", "disabled"),
				},
			},
			"host_group_ids": schema.SetAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Set of host group IDs the host belongs to. At least one is required. This set is authoritative — any groups not listed here are removed on apply.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"tags": schema.SetNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Set of tags to attach to the host.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Tag name.",
						},
						"value": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Tag value.",
						},
					},
				},
			},
			"inventory_mode": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("disabled"),
				MarkdownDescription: "Inventory population mode. One of: `disabled`, `manual`, `automatic`. Defaults to `disabled`.",
				Validators: []validator.String{
					stringvalidator.OneOf("disabled", "manual", "automatic"),
				},
			},
			"proxy_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("0"),
				MarkdownDescription: "ID of the proxy that monitors the host. Set to `0` for no proxy.",
			},
		},
	}
}

func (r *HostResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *HostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data HostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	h, diags := modelToClientHost(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := client.HostCreate(ctx, r.client, h)
	if err != nil {
		resp.Diagnostics.AddError("Error creating host", err.Error())
		return
	}
	data.ID = types.StringValue(id)

	created, err := client.HostGet(ctx, r.client, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading host after create", err.Error())
		return
	}
	if created != nil {
		resp.Diagnostics.Append(clientHostToModel(ctx, *created, &data)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data HostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	h, err := client.HostGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading host", err.Error())
		return
	}
	if h == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(clientHostToModel(ctx, *h, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data HostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	h, diags := modelToClientHost(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.HostUpdate(ctx, r.client, h); err != nil {
		resp.Diagnostics.AddError("Error updating host", err.Error())
		return
	}

	updated, err := client.HostGet(ctx, r.client, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading host after update", err.Error())
		return
	}
	if updated != nil {
		resp.Diagnostics.Append(clientHostToModel(ctx, *updated, &data)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data HostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.HostDelete(ctx, r.client, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting host", err.Error())
		return
	}
}

func (r *HostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// modelToClientHost converts a HostResourceModel to a client.Host for API calls.
func modelToClientHost(ctx context.Context, data HostResourceModel) (client.Host, diag.Diagnostics) {
	var diags diag.Diagnostics

	var groupIDStrings []string
	diags.Append(data.HostGroupIDs.ElementsAs(ctx, &groupIDStrings, false)...)
	if diags.HasError() {
		return client.Host{}, diags
	}
	groups := make([]client.HostGroupRef, len(groupIDStrings))
	for i, id := range groupIDStrings {
		groups[i] = client.HostGroupRef{GroupID: id}
	}

	var tagModels []HostTagModel
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		diags.Append(data.Tags.ElementsAs(ctx, &tagModels, false)...)
		if diags.HasError() {
			return client.Host{}, diags
		}
	}
	tags := make([]client.HostTag, len(tagModels))
	for i, t := range tagModels {
		tags[i] = client.HostTag{Tag: t.Name.ValueString(), Value: t.Value.ValueString()}
	}

	return client.Host{
		HostID:        data.ID.ValueString(),
		Host:          data.Host.ValueString(),
		Name:          data.Name.ValueString(),
		Description:   data.Description.ValueString(),
		Status:        hostStatusMap[data.Status.ValueString()],
		Groups:        groups,
		Tags:          tags,
		InventoryMode: hostInventoryModeMap[data.InventoryMode.ValueString()],
		ProxyID:       data.ProxyID.ValueString(),
	}, diags
}

// clientHostToModel updates a HostResourceModel from a client.Host response.
func clientHostToModel(_ context.Context, h client.Host, data *HostResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.Host = types.StringValue(h.Host)
	data.Name = types.StringValue(h.Name)
	data.Description = types.StringValue(h.Description)
	data.Status = types.StringValue(hostStatusReverseMap[h.Status])
	data.InventoryMode = types.StringValue(hostInventoryModeReverseMap[h.InventoryMode])
	data.ProxyID = types.StringValue(h.ProxyID)

	groupIDVals := make([]attr.Value, len(h.Groups))
	for i, g := range h.Groups {
		groupIDVals[i] = types.StringValue(g.GroupID)
	}
	groupSet, d := types.SetValue(types.StringType, groupIDVals)
	diags.Append(d...)
	if !d.HasError() {
		data.HostGroupIDs = groupSet
	}

	tagVals := make([]attr.Value, len(h.Tags))
	for i, t := range h.Tags {
		obj, d := types.ObjectValue(hostTagAttrTypes, map[string]attr.Value{
			"name":  types.StringValue(t.Tag),
			"value": types.StringValue(t.Value),
		})
		diags.Append(d...)
		if d.HasError() {
			return diags
		}
		tagVals[i] = obj
	}
	tagSet, d := types.SetValue(types.ObjectType{AttrTypes: hostTagAttrTypes}, tagVals)
	diags.Append(d...)
	if !d.HasError() {
		data.Tags = tagSet
	}

	return diags
}
