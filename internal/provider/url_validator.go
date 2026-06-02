package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// URLValidator is a string validator that requires a valid http/https URL.
type URLValidator struct{}

func (v URLValidator) Description(_ context.Context) string {
	return "value must be a valid absolute URL with http or https scheme"
}

func (v URLValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v URLValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	raw := req.ConfigValue.ValueString()
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid URL",
			fmt.Sprintf("%q must be an absolute URL with http or https scheme (e.g. https://zabbix.example.com).", raw),
		)
	}
}
