package provider_test

import (
	"context"
	"testing"

	"github.com/gringolito/terraform-provider-zabbix/internal/provider"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestMediaTypeEmailResource_EnumValidation verifies that smtp_security,
// smtp_authentication, content_type, status, event_source, and recovery all
// reject invalid strings at plan time via stringvalidator.OneOf.
func TestMediaTypeEmailResource_EnumValidation(t *testing.T) {
	r := provider.NewMediaTypeEmailResource()
	schResp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, schResp)

	nestedList, ok := schResp.Schema.Attributes["message_templates"].(fwschema.ListNestedAttribute)
	if !ok {
		t.Fatal("attribute \"message_templates\" is not a ListNestedAttribute")
	}
	nestedAttrs := nestedList.NestedObject.Attributes

	cases := []struct {
		attr    string
		from    map[string]fwschema.Attribute
		valid   []string
		invalid string
	}{
		{
			attr:    "smtp_security",
			from:    schResp.Schema.Attributes,
			valid:   []string{"none", "starttls", "ssl_tls"},
			invalid: "ssl",
		},
		{
			attr:    "smtp_authentication",
			from:    schResp.Schema.Attributes,
			valid:   []string{"none", "normal_password"},
			invalid: "password",
		},
		{
			attr:    "content_type",
			from:    schResp.Schema.Attributes,
			valid:   []string{"text", "html"},
			invalid: "plain",
		},
		{
			attr:    "status",
			from:    schResp.Schema.Attributes,
			valid:   []string{"enabled", "disabled"},
			invalid: "active",
		},
		{
			attr:    "event_source",
			from:    nestedAttrs,
			valid:   []string{"trigger", "discovery", "autoregistration", "internal", "service"},
			invalid: "alarm",
		},
		{
			attr:    "recovery",
			from:    nestedAttrs,
			valid:   []string{"operation", "recovery", "update"},
			invalid: "resolved",
		},
	}

	for _, tc := range cases {
		t.Run(tc.attr, func(t *testing.T) {
			attr, ok := tc.from[tc.attr].(fwschema.StringAttribute)
			if !ok {
				t.Fatalf("attribute %q is not a StringAttribute", tc.attr)
			}
			if len(attr.Validators) == 0 {
				t.Fatalf("attribute %q has no validators; add stringvalidator.OneOf", tc.attr)
			}

			for _, v := range tc.valid {
				req := validator.StringRequest{ConfigValue: types.StringValue(v)}
				resp := &validator.StringResponse{}
				for _, val := range attr.Validators {
					val.ValidateString(context.Background(), req, resp)
				}
				if resp.Diagnostics.HasError() {
					t.Errorf("attribute %q should accept %q but got error", tc.attr, v)
				}
			}

			req := validator.StringRequest{ConfigValue: types.StringValue(tc.invalid)}
			resp := &validator.StringResponse{}
			for _, val := range attr.Validators {
				val.ValidateString(context.Background(), req, resp)
			}
			if !resp.Diagnostics.HasError() {
				t.Errorf("attribute %q should reject %q", tc.attr, tc.invalid)
			}
		})
	}
}
