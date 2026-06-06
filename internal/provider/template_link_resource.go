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

var _ resource.Resource = &TemplateLinkResource{}
var _ resource.ResourceWithImportState = &TemplateLinkResource{}

func NewTemplateLinkResource() resource.Resource {
	return &TemplateLinkResource{}
}

type TemplateLinkResource struct {
	client client.Client
}

type TemplateLinkResourceModel struct {
	ID               types.String `tfsdk:"id"`
	TemplateID       types.String `tfsdk:"template_id"`
	LinkedTemplateID types.String `tfsdk:"linked_template_id"`
	OnDestroy        types.String `tfsdk:"on_destroy"`
}

func (r *TemplateLinkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template_link"
}

func (r *TemplateLinkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Links a parent template to a child template. The child template inherits items, triggers, and graphs from the parent.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier in the form `<template_id>/<linked_template_id>`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"template_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the child template that inherits from the parent.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"linked_template_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the parent template to inherit from.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"on_destroy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("clear"),
				MarkdownDescription: "Behaviour on destroy: `clear` (default) removes inherited items/triggers/graphs from the child; `unlink` unlinks only, leaving inherited entities in place.",
				Validators: []validator.String{
					stringvalidator.OneOf("clear", "unlink"),
				},
			},
		},
	}
}

func (r *TemplateLinkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TemplateLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TemplateLinkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.TemplateLinkAdd(ctx, r.client, data.TemplateID.ValueString(), []string{data.LinkedTemplateID.ValueString()}); err != nil {
		resp.Diagnostics.AddError("Error creating template link", err.Error())
		return
	}

	data.ID = types.StringValue(data.TemplateID.ValueString() + "/" + data.LinkedTemplateID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TemplateLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TemplateLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	t, err := client.TemplateGet(ctx, r.client, data.TemplateID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading template", err.Error())
		return
	}
	if t == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	linked := false
	for _, ref := range t.ParentTemplates {
		if ref.TemplateID == data.LinkedTemplateID.ValueString() {
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

func (r *TemplateLinkResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All fields are RequiresReplace — Update is never called.
}

func (r *TemplateLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TemplateLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	doClear := data.OnDestroy.ValueString() != "unlink"
	if err := client.TemplateLinkRemove(ctx, r.client, data.TemplateID.ValueString(), data.LinkedTemplateID.ValueString(), doClear); err != nil {
		resp.Diagnostics.AddError("Error deleting template link", err.Error())
	}
}

func (r *TemplateLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format <template_id>/<linked_template_id>, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &TemplateLinkResourceModel{
		ID:               types.StringValue(req.ID),
		TemplateID:       types.StringValue(parts[0]),
		LinkedTemplateID: types.StringValue(parts[1]),
		OnDestroy:        types.StringValue("clear"),
	})...)
}
