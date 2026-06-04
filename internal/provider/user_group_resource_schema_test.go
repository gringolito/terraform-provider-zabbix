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

// TestUserGroupResource_EnumValidation verifies that gui_access, debug_mode, and
// users_status reject unknown strings at plan time via stringvalidator.OneOf.
func TestUserGroupResource_EnumValidation(t *testing.T) {
	r := provider.NewUserGroupResource()
	schResp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, schResp)

	cases := []struct {
		attr    string
		valid   []string
		invalid string
	}{
		{"gui_access", []string{"system_default", "internal", "disabled"}, "wrong"},
		{"debug_mode", []string{"disabled", "enabled"}, "wrong"},
		{"users_status", []string{"enabled", "disabled"}, "wrong"},
	}

	for _, tc := range cases {
		t.Run(tc.attr, func(t *testing.T) {
			attr, ok := schResp.Schema.Attributes[tc.attr].(fwschema.StringAttribute)
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
