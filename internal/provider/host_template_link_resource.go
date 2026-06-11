package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &HostTemplateLinkResource{}
var _ resource.ResourceWithImportState = &HostTemplateLinkResource{}

func NewHostTemplateLinkResource() resource.Resource {
	return &HostTemplateLinkResource{}
}

type HostTemplateLinkResource struct {
	client client.Client
}

type HostTemplateLinkResourceModel struct {
	ID         types.String `tfsdk:"id"`
	HostID     types.String `tfsdk:"host_id"`
	TemplateID types.String `tfsdk:"template_id"`
	OnDestroy  types.String `tfsdk:"on_destroy"`
}

func (r *HostTemplateLinkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host_template_link"
}

func (r *HostTemplateLinkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Links a template to a host. The host inherits items, triggers, and graphs from the template.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier in the form `<host_id>/<template_id>`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"host_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the host that inherits from the template.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"template_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the template to link to the host.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"on_destroy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("clear"),
				MarkdownDescription: "Behaviour on destroy: `clear` (default) removes inherited items/triggers/graphs from the host; `unlink` unlinks only, leaving inherited entities in place as host-local items.",
				Validators: []validator.String{
					stringvalidator.OneOf("clear", "unlink"),
				},
			},
		},
	}
}

func (r *HostTemplateLinkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *HostTemplateLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data HostTemplateLinkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.HostTemplateLinkAdd(ctx, r.client, data.HostID.ValueString(), []string{data.TemplateID.ValueString()}); err != nil {
		resp.Diagnostics.AddError("Error creating host template link", err.Error())
		return
	}

	data.ID = types.StringValue(data.HostID.ValueString() + "/" + data.TemplateID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HostTemplateLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data HostTemplateLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	templates, err := client.HostGetTemplates(ctx, r.client, data.HostID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading host templates", err.Error())
		return
	}
	if templates == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	linked := false
	for _, ref := range templates {
		if ref.TemplateID == data.TemplateID.ValueString() {
			linked = true
			break
		}
	}
	if !linked {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HostTemplateLinkResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All fields are RequiresReplace — Update is never called.
}

func (r *HostTemplateLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data HostTemplateLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	doClear := data.OnDestroy.ValueString() != "unlink"
	if err := client.HostTemplateLinkRemove(ctx, r.client, data.HostID.ValueString(), data.TemplateID.ValueString(), doClear); err != nil {
		resp.Diagnostics.AddError("Error deleting host template link", err.Error())
	}
}

func (r *HostTemplateLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format <host_id>/<template_id>, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &HostTemplateLinkResourceModel{
		ID:         types.StringValue(req.ID),
		HostID:     types.StringValue(parts[0]),
		TemplateID: types.StringValue(parts[1]),
		OnDestroy:  types.StringValue("clear"),
	})...)
}
