package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/mail"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	accountNameRegexp  = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
	bucketNameRegexp   = regexp.MustCompile(`^[a-z0-9][a-z0-9.\-]*[a-z0-9]$`)
	iamNameRegexp      = regexp.MustCompile(`^[\w+=,.@-]+$`)
	hostnamePartRegexp = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
)

// AccountName validates ARTESCA account names: 1–128 ASCII alphanumeric characters and hyphens.
type AccountName struct{}

func (v AccountName) Description(_ context.Context) string {
	return "must be 1–128 characters, ASCII alphanumeric and hyphens only"
}

func (v AccountName) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v AccountName) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if len(val) < 1 || len(val) > 128 {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid account name",
			fmt.Sprintf("Must be 1–128 characters, got %d.", len(val)))
		return
	}
	if !accountNameRegexp.MatchString(val) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid account name",
			"Must contain only ASCII alphanumeric characters and hyphens.")
	}
}

// BucketName validates S3 bucket names: 3–63 characters, lowercase letters, numbers, hyphens, and periods.
type BucketName struct{}

func (v BucketName) Description(_ context.Context) string {
	return "must be 3–63 characters, lowercase letters, numbers, hyphens, and periods"
}

func (v BucketName) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v BucketName) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if len(val) < 3 || len(val) > 63 {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid bucket name",
			fmt.Sprintf("Must be 3–63 characters, got %d.", len(val)))
		return
	}
	if !bucketNameRegexp.MatchString(val) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid bucket name",
			"Must contain only lowercase letters, numbers, hyphens, and periods, and must start and end with a letter or number.")
	}
}

// IAMName validates IAM usernames and policy names per AWS spec.
type IAMName struct {
	MaxLength int
	FieldName string
}

func (v IAMName) Description(_ context.Context) string {
	return fmt.Sprintf("must be 1–%d characters, alphanumeric and [+=,.@-]", v.MaxLength)
}

func (v IAMName) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v IAMName) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if len(val) < 1 || len(val) > v.MaxLength {
		resp.Diagnostics.AddAttributeError(req.Path, fmt.Sprintf("Invalid %s", v.FieldName),
			fmt.Sprintf("Must be 1–%d characters, got %d.", v.MaxLength, len(val)))
		return
	}
	if !iamNameRegexp.MatchString(val) {
		resp.Diagnostics.AddAttributeError(req.Path, fmt.Sprintf("Invalid %s", v.FieldName),
			"Must contain only alphanumeric characters and the characters +=,.@-.")
	}
}

// IAMUsername returns a validator for IAM usernames (1–64 chars).
func IAMUsername() IAMName {
	return IAMName{MaxLength: 64, FieldName: "IAM username"}
}

// IAMPolicyName returns a validator for IAM policy names (1–128 chars).
func IAMPolicyName() IAMName {
	return IAMName{MaxLength: 128, FieldName: "IAM policy name"}
}

// Email validates email addresses using Go's net/mail parser.
type Email struct{}

func (v Email) Description(_ context.Context) string {
	return "must be a valid email address"
}

func (v Email) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v Email) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if val == "" {
		return
	}
	_, err := mail.ParseAddress(val)
	if err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid email address",
			fmt.Sprintf("Must be a valid email address: %s.", err))
	}
}

// Hostname validates DNS hostnames and IP addresses (IPv4 and IPv6).
type Hostname struct{}

func (v Hostname) Description(_ context.Context) string {
	return "must be a valid DNS hostname or IP address"
}

func (v Hostname) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v Hostname) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if val == "" {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid hostname", "Must not be empty.")
		return
	}

	if net.ParseIP(val) != nil {
		return
	}

	trimmed := strings.TrimSuffix(val, ".")
	if len(trimmed) > 253 {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid hostname",
			"Hostname must not exceed 253 characters.")
		return
	}

	parts := strings.Split(trimmed, ".")
	for _, part := range parts {
		if len(part) == 0 || len(part) > 63 {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid hostname",
				fmt.Sprintf("Each label must be 1–63 characters, got %d.", len(part)))
			return
		}
		if !hostnamePartRegexp.MatchString(part) {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid hostname",
				fmt.Sprintf("Label %q contains invalid characters.", part))
			return
		}
	}
}

// JSONDocument validates that a string is valid JSON.
type JSONDocument struct{}

func (v JSONDocument) Description(_ context.Context) string {
	return "must be a valid JSON document"
}

func (v JSONDocument) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v JSONDocument) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if !json.Valid([]byte(val)) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid JSON",
			"Must be a valid JSON document.")
	}
}

// SSEAlgorithm validates that the value is one of the SSE algorithms ARTESCA accepts.
type SSEAlgorithm struct{}

func (v SSEAlgorithm) Description(_ context.Context) string {
	return `must be "AES256"`
}

func (v SSEAlgorithm) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v SSEAlgorithm) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if val != "AES256" {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid SSE algorithm",
			fmt.Sprintf("Must be \"AES256\", got %q.", val))
	}
}
