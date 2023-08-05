package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = ttlAtLeast600Validator{}

type ttlAtLeast600Validator struct{}

// Description describes the validation in plain text formatting.
func (validator ttlAtLeast600Validator) Description(_ context.Context) string {
	return fmt.Sprintf("ttl must be at least 600")
}

// MarkdownDescription describes the validation in Markdown formatting.
func (validator ttlAtLeast600Validator) MarkdownDescription(ctx context.Context) string {
	return validator.Description(ctx)
}

// Validate runs the main validation logic of the validator, reading configuration data out of `req` and updating `resp` with diagnostics.
func (v ttlAtLeast600Validator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	// If the value is unknown or null, there is nothing to validate.
	if request.ConfigValue.IsUnknown() || request.ConfigValue.IsNull() {
		return
	}

	ttlInt, err := strconv.Atoi(request.ConfigValue.ValueString())
	if err != nil {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"invalid value for ttl",
			fmt.Sprint(err),
		)
		return
	}

	if ttlInt < 600 {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"invalid value for ttl",
			fmt.Sprintf("provided ttl %v is less than 600", ttlInt),
		)
		return

	}
}

func TtlAtLeast600() validator.String {
	return ttlAtLeast600Validator{}
}
