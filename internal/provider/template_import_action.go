package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-zabbix/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = &TemplateImportAction{}
var _ action.ActionWithConfigure = &TemplateImportAction{}

func NewTemplateImportAction() action.Action {
	return &TemplateImportAction{}
}

type TemplateImportAction struct {
	client client.Client
}

type templateImportActionModel struct {
	Source types.String      `tfsdk:"source"`
	Format types.String      `tfsdk:"format"`
	Rules  *importRulesModel `tfsdk:"rules"`
}

type importRulesModel struct {
	Templates          *importRuleCreateUpdateModel `tfsdk:"templates"`
	TemplateGroups     *importRuleCreateUpdateModel `tfsdk:"template_groups"`
	TemplateLinkage    *importRuleCreateDeleteModel `tfsdk:"template_linkage"`
	DiscoveryRules     *importRuleAllModel          `tfsdk:"discovery_rules"`
	Graphs             *importRuleAllModel          `tfsdk:"graphs"`
	HTTPTests          *importRuleAllModel          `tfsdk:"http_tests"`
	Items              *importRuleAllModel          `tfsdk:"items"`
	TemplateDashboards *importRuleAllModel          `tfsdk:"template_dashboards"`
	Triggers           *importRuleAllModel          `tfsdk:"triggers"`
	ValueMaps          *importRuleAllModel          `tfsdk:"value_maps"`
}

type importRuleCreateUpdateModel struct {
	CreateMissing  types.Bool `tfsdk:"create_missing"`
	UpdateExisting types.Bool `tfsdk:"update_existing"`
}

type importRuleCreateDeleteModel struct {
	CreateMissing types.Bool `tfsdk:"create_missing"`
	DeleteMissing types.Bool `tfsdk:"delete_missing"`
}

type importRuleAllModel struct {
	CreateMissing  types.Bool `tfsdk:"create_missing"`
	UpdateExisting types.Bool `tfsdk:"update_existing"`
	DeleteMissing  types.Bool `tfsdk:"delete_missing"`
}

func (a *TemplateImportAction) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template_import"
}

func (a *TemplateImportAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	createUpdate := map[string]schema.Attribute{
		"create_missing":  schema.BoolAttribute{Optional: true},
		"update_existing": schema.BoolAttribute{Optional: true},
	}
	createDelete := map[string]schema.Attribute{
		"create_missing": schema.BoolAttribute{Optional: true},
		"delete_missing": schema.BoolAttribute{Optional: true},
	}
	createUpdateDelete := map[string]schema.Attribute{
		"create_missing":  schema.BoolAttribute{Optional: true},
		"update_existing": schema.BoolAttribute{Optional: true},
		"delete_missing":  schema.BoolAttribute{Optional: true},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Imports Zabbix templates from an export file (XML, YAML, or JSON) via the " +
			"`configuration.import` API. This is a stateless action: it provides no drift detection and " +
			"no cleanup — imported templates remain in Zabbix after `terraform destroy`. Reference " +
			"imported templates via the `zabbix_template` data source.\n\n" +
			"The `source` attribute accepts raw content; compose it with the `file()` function or the " +
			"`http` data source.",
		Attributes: map[string]schema.Attribute{
			"source": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Raw XML, YAML, or JSON content of a Zabbix template export. Use `file(\"path/to/export.xml\")` or the `http` data source to supply the content.",
			},
			"format": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Format of the export content. Must match the actual content format. Accepted values: `xml`, `yaml`, `json`.",
				Validators: []validator.String{
					stringvalidator.OneOf("xml", "yaml", "json"),
				},
			},
			"rules": schema.SingleNestedAttribute{
				Optional: true,
				MarkdownDescription: "Import rules controlling which entities are created, updated, or deleted. " +
					"Omit to use the defaults: `create_missing=true`, `update_existing=true`, `delete_missing=false` for all entities.",
				Attributes: map[string]schema.Attribute{
					"templates": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Rules for template objects.",
						Attributes:          createUpdate,
					},
					"template_groups": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Rules for template group objects.",
						Attributes:          createUpdate,
					},
					"template_linkage": schema.SingleNestedAttribute{
						Optional: true,
						MarkdownDescription: "Rules for template linkage. " +
							"`delete_missing=true` unlinks parent templates that are absent in the import source, without deleting inherited entities.",
						Attributes: createDelete,
					},
					"discovery_rules": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Rules for low-level discovery rule objects.",
						Attributes:          createUpdateDelete,
					},
					"graphs": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Rules for graph objects.",
						Attributes:          createUpdateDelete,
					},
					"http_tests": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Rules for HTTP test (web scenario) objects.",
						Attributes:          createUpdateDelete,
					},
					"items": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Rules for item objects.",
						Attributes:          createUpdateDelete,
					},
					"template_dashboards": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Rules for template dashboard objects.",
						Attributes:          createUpdateDelete,
					},
					"triggers": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Rules for trigger objects.",
						Attributes:          createUpdateDelete,
					},
					"value_maps": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Rules for value map objects.",
						Attributes:          createUpdateDelete,
					},
				},
			},
		},
	}
}

func (a *TemplateImportAction) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Action Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	a.client = c
}

func (a *TemplateImportAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	resp.SendProgress(action.InvokeProgressEvent{Message: "importing Zabbix templates"})

	var data templateImportActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rules := toClientRules(data.Rules)

	if err := client.ConfigurationImport(ctx, a.client, data.Format.ValueString(), data.Source.ValueString(), rules); err != nil {
		resp.Diagnostics.AddError("Template Import Failed", fmt.Sprintf("configuration.import: %s", err))
		return
	}

	resp.SendProgress(action.InvokeProgressEvent{Message: "template import complete"})
}

func toClientRules(m *importRulesModel) client.ImportRules {
	rules := defaultImportRules()
	if m == nil {
		return rules
	}
	if m.Templates != nil {
		rules.Templates.CreateMissing = boolVal(m.Templates.CreateMissing, true)
		rules.Templates.UpdateExisting = boolVal(m.Templates.UpdateExisting, true)
	}
	if m.TemplateGroups != nil {
		rules.TemplateGroups.CreateMissing = boolVal(m.TemplateGroups.CreateMissing, true)
		rules.TemplateGroups.UpdateExisting = boolVal(m.TemplateGroups.UpdateExisting, true)
	}
	if m.TemplateLinkage != nil {
		rules.TemplateLinkage.CreateMissing = boolVal(m.TemplateLinkage.CreateMissing, true)
		rules.TemplateLinkage.DeleteMissing = boolVal(m.TemplateLinkage.DeleteMissing, false)
	}
	if m.DiscoveryRules != nil {
		rules.DiscoveryRules = importRuleAllFromModel(m.DiscoveryRules)
	}
	if m.Graphs != nil {
		rules.Graphs = importRuleAllFromModel(m.Graphs)
	}
	if m.HTTPTests != nil {
		rules.HTTPTests = importRuleAllFromModel(m.HTTPTests)
	}
	if m.Items != nil {
		rules.Items = importRuleAllFromModel(m.Items)
	}
	if m.TemplateDashboards != nil {
		rules.TemplateDashboards = importRuleAllFromModel(m.TemplateDashboards)
	}
	if m.Triggers != nil {
		rules.Triggers = importRuleAllFromModel(m.Triggers)
	}
	if m.ValueMaps != nil {
		rules.ValueMaps = importRuleAllFromModel(m.ValueMaps)
	}
	return rules
}

func importRuleAllFromModel(m *importRuleAllModel) client.ImportRuleAll {
	return client.ImportRuleAll{
		CreateMissing:  boolVal(m.CreateMissing, true),
		UpdateExisting: boolVal(m.UpdateExisting, true),
		DeleteMissing:  boolVal(m.DeleteMissing, false),
	}
}

func boolVal(v types.Bool, def bool) bool {
	if v.IsNull() || v.IsUnknown() {
		return def
	}
	return v.ValueBool()
}

func defaultImportRules() client.ImportRules {
	return client.ImportRules{
		Templates:          client.ImportRuleCreateUpdate{CreateMissing: true, UpdateExisting: true},
		TemplateGroups:     client.ImportRuleCreateUpdate{CreateMissing: true, UpdateExisting: true},
		TemplateLinkage:    client.ImportRuleCreateDelete{CreateMissing: true, DeleteMissing: false},
		DiscoveryRules:     client.ImportRuleAll{CreateMissing: true, UpdateExisting: true, DeleteMissing: false},
		Graphs:             client.ImportRuleAll{CreateMissing: true, UpdateExisting: true, DeleteMissing: false},
		HTTPTests:          client.ImportRuleAll{CreateMissing: true, UpdateExisting: true, DeleteMissing: false},
		Items:              client.ImportRuleAll{CreateMissing: true, UpdateExisting: true, DeleteMissing: false},
		TemplateDashboards: client.ImportRuleAll{CreateMissing: true, UpdateExisting: true, DeleteMissing: false},
		Triggers:           client.ImportRuleAll{CreateMissing: true, UpdateExisting: true, DeleteMissing: false},
		ValueMaps:          client.ImportRuleAll{CreateMissing: true, UpdateExisting: true, DeleteMissing: false},
	}
}
