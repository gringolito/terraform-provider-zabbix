package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TagModel represents a single Zabbix {name, value} tag.
// Zabbix allows multiple tags with the same name but different values, so tags
// are always modeled as a set of objects rather than a map — see ADR 0011.
type TagModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

var tagAttrTypes = map[string]attr.Type{
	"name":  types.StringType,
	"value": types.StringType,
}
