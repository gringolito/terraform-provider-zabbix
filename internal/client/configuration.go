package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// ImportRuleCreateUpdate holds rules for entities that support createMissing and updateExisting.
type ImportRuleCreateUpdate struct {
	CreateMissing  bool `json:"createMissing"`
	UpdateExisting bool `json:"updateExisting"`
}

// ImportRuleCreateDelete holds rules for entities that support createMissing and deleteMissing.
// deleteMissing for templateLinkage unlinks templates without deleting inherited entities.
type ImportRuleCreateDelete struct {
	CreateMissing bool `json:"createMissing"`
	DeleteMissing bool `json:"deleteMissing"`
}

// ImportRuleAll holds rules for entities that support all three toggles.
type ImportRuleAll struct {
	CreateMissing  bool `json:"createMissing"`
	UpdateExisting bool `json:"updateExisting"`
	DeleteMissing  bool `json:"deleteMissing"`
}

// ImportRules maps to the Zabbix 7.0 configuration.import rules parameter.
type ImportRules struct {
	Templates          ImportRuleCreateUpdate `json:"templates"`
	TemplateGroups     ImportRuleCreateUpdate `json:"template_groups"`
	TemplateLinkage    ImportRuleCreateDelete `json:"templateLinkage"`
	DiscoveryRules     ImportRuleAll          `json:"discoveryRules"`
	Graphs             ImportRuleAll          `json:"graphs"`
	HTTPTests          ImportRuleAll          `json:"httptests"`
	Items              ImportRuleAll          `json:"items"`
	TemplateDashboards ImportRuleAll          `json:"templateDashboards"`
	Triggers           ImportRuleAll          `json:"triggers"`
	ValueMaps          ImportRuleAll          `json:"valueMaps"`
}

// ConfigurationImport calls configuration.import to load templates from an export blob.
// format must be one of "xml", "yaml", or "json".
func ConfigurationImport(ctx context.Context, c Client, format, source string, rules ImportRules) error {
	params := map[string]any{
		"format": format,
		"source": source,
		"rules":  rules,
	}
	result, err := c.Call(ctx, "configuration.import", params)
	if err != nil {
		return err
	}
	var ok bool
	if err := json.Unmarshal(result, &ok); err != nil {
		return fmt.Errorf("configuration.import: unexpected response: %w", err)
	}
	if !ok {
		return fmt.Errorf("configuration.import: server returned false")
	}
	return nil
}
