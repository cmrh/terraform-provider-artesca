package validators

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func testStringRequest(val string) validator.StringRequest {
	return validator.StringRequest{
		Path:        path.Root("test"),
		ConfigValue: types.StringValue(val),
	}
}

func testNullRequest() validator.StringRequest {
	return validator.StringRequest{
		Path:        path.Root("test"),
		ConfigValue: types.StringNull(),
	}
}

func testUnknownRequest() validator.StringRequest {
	return validator.StringRequest{
		Path:        path.Root("test"),
		ConfigValue: types.StringUnknown(),
	}
}

func TestAccountName(t *testing.T) {
	v := AccountName{}
	ctx := context.Background()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid simple", "my-account", false},
		{"valid alphanumeric", "account123", false},
		{"valid single char", "a", false},
		{"valid 128 chars", strings.Repeat("a", 128), false},
		{"valid hyphens", "my-cool-account", false},
		{"valid uppercase", "MyAccount", false},
		{"invalid empty", "", true},
		{"invalid 129 chars", strings.Repeat("a", 129), true},
		{"invalid spaces", "my account", true},
		{"invalid underscore", "my_account", true},
		{"invalid special chars", "account!", true},
		{"invalid dot", "my.account", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			v.ValidateString(ctx, testStringRequest(tt.value), resp)
			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %q, got none", tt.value)
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("expected no error for %q, got: %s", tt.value, resp.Diagnostics.Errors())
			}
		})
	}
}

func TestAccountName_NullAndUnknown(t *testing.T) {
	v := AccountName{}
	ctx := context.Background()

	resp := &validator.StringResponse{}
	v.ValidateString(ctx, testNullRequest(), resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for null value")
	}

	resp = &validator.StringResponse{}
	v.ValidateString(ctx, testUnknownRequest(), resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for unknown value")
	}
}

func TestBucketName(t *testing.T) {
	v := BucketName{}
	ctx := context.Background()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid simple", "my-bucket", false},
		{"valid with dots", "my.bucket.name", false},
		{"valid 3 chars", "abc", false},
		{"valid 63 chars", strings.Repeat("a", 63), false},
		{"valid numbers", "123bucket", false},
		{"valid complex", "my-bucket.v2.data", false},
		{"invalid too short", "ab", true},
		{"invalid too long", strings.Repeat("a", 64), true},
		{"invalid uppercase", "MyBucket", true},
		{"invalid start hyphen", "-bucket", true},
		{"invalid end hyphen", "bucket-", true},
		{"invalid start dot", ".bucket", true},
		{"invalid underscore", "my_bucket", true},
		{"invalid spaces", "my bucket", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			v.ValidateString(ctx, testStringRequest(tt.value), resp)
			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %q, got none", tt.value)
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("expected no error for %q, got: %s", tt.value, resp.Diagnostics.Errors())
			}
		})
	}
}

func TestBucketName_NullAndUnknown(t *testing.T) {
	v := BucketName{}
	ctx := context.Background()

	resp := &validator.StringResponse{}
	v.ValidateString(ctx, testNullRequest(), resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for null value")
	}

	resp = &validator.StringResponse{}
	v.ValidateString(ctx, testUnknownRequest(), resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for unknown value")
	}
}

func TestIAMUsername(t *testing.T) {
	v := IAMUsername()
	ctx := context.Background()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid simple", "myuser", false},
		{"valid with special", "user+=,.@-name", false},
		{"valid single char", "a", false},
		{"valid 64 chars", strings.Repeat("a", 64), false},
		{"valid underscore", "my_user", false},
		{"invalid empty", "", true},
		{"invalid 65 chars", strings.Repeat("a", 65), true},
		{"invalid spaces", "my user", true},
		{"invalid exclamation", "user!", true},
		{"invalid colon", "user:name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			v.ValidateString(ctx, testStringRequest(tt.value), resp)
			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %q, got none", tt.value)
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("expected no error for %q, got: %s", tt.value, resp.Diagnostics.Errors())
			}
		})
	}
}

func TestIAMPolicyName(t *testing.T) {
	v := IAMPolicyName()
	ctx := context.Background()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid simple", "MyPolicy", false},
		{"valid 128 chars", strings.Repeat("a", 128), false},
		{"valid with special", "policy+=,.@-", false},
		{"invalid empty", "", true},
		{"invalid 129 chars", strings.Repeat("a", 129), true},
		{"invalid spaces", "my policy", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			v.ValidateString(ctx, testStringRequest(tt.value), resp)
			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %q, got none", tt.value)
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("expected no error for %q, got: %s", tt.value, resp.Diagnostics.Errors())
			}
		})
	}
}

func TestEmail(t *testing.T) {
	v := Email{}
	ctx := context.Background()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid simple", "user@example.com", false},
		{"valid with plus", "user+tag@example.com", false},
		{"valid subdomain", "user@sub.example.com", false},
		{"valid empty string", "", false},
		{"invalid no at", "userexample.com", true},
		{"invalid no domain", "user@", true},
		{"invalid double at", "user@@example.com", true},
		{"invalid bare word", "notanemail", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			v.ValidateString(ctx, testStringRequest(tt.value), resp)
			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %q, got none", tt.value)
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("expected no error for %q, got: %s", tt.value, resp.Diagnostics.Errors())
			}
		})
	}
}

func TestEmail_NullAndUnknown(t *testing.T) {
	v := Email{}
	ctx := context.Background()

	resp := &validator.StringResponse{}
	v.ValidateString(ctx, testNullRequest(), resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for null value")
	}

	resp = &validator.StringResponse{}
	v.ValidateString(ctx, testUnknownRequest(), resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for unknown value")
	}
}

func TestHostname(t *testing.T) {
	v := Hostname{}
	ctx := context.Background()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid domain", "mybucket.example.com", false},
		{"valid short domain", "example.com", false},
		{"valid single label", "localhost", false},
		{"valid trailing dot", "example.com.", false},
		{"valid ipv4", "192.168.1.1", false},
		{"valid ipv4 loopback", "127.0.0.1", false},
		{"valid ipv6", "::1", false},
		{"valid ipv6 full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", false},
		{"valid ipv6 short", "2001:db8::1", false},
		{"valid hyphen in label", "my-host.example.com", false},
		{"invalid empty", "", true},
		{"invalid start hyphen label", "-host.example.com", true},
		{"invalid end hyphen label", "host-.example.com", true},
		{"invalid underscore", "my_host.example.com", true},
		{"invalid label too long", strings.Repeat("a", 64) + ".com", true},
		{"invalid hostname too long", strings.Repeat("a", 60) + "." + strings.Repeat(strings.Repeat("a", 60)+".", 4) + "com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			v.ValidateString(ctx, testStringRequest(tt.value), resp)
			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %q, got none", tt.value)
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("expected no error for %q, got: %s", tt.value, resp.Diagnostics.Errors())
			}
		})
	}
}

func TestHostname_NullAndUnknown(t *testing.T) {
	v := Hostname{}
	ctx := context.Background()

	resp := &validator.StringResponse{}
	v.ValidateString(ctx, testNullRequest(), resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for null value")
	}

	resp = &validator.StringResponse{}
	v.ValidateString(ctx, testUnknownRequest(), resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for unknown value")
	}
}

func TestJSONDocument(t *testing.T) {
	v := JSONDocument{}
	ctx := context.Background()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid object", `{"Version":"2012-10-17","Statement":[]}`, false},
		{"valid array", `[1,2,3]`, false},
		{"valid string", `"hello"`, false},
		{"invalid malformed", `{not json}`, true},
		{"invalid trailing comma", `{"a":1,}`, true},
		{"invalid empty", ``, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			v.ValidateString(ctx, testStringRequest(tt.value), resp)
			if tt.wantErr && !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %q, got none", tt.value)
			}
			if !tt.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("expected no error for %q, got: %s", tt.value, resp.Diagnostics.Errors())
			}
		})
	}
}

func TestJSONDocument_NullAndUnknown(t *testing.T) {
	v := JSONDocument{}
	ctx := context.Background()

	resp := &validator.StringResponse{}
	v.ValidateString(ctx, testNullRequest(), resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for null value")
	}

	resp = &validator.StringResponse{}
	v.ValidateString(ctx, testUnknownRequest(), resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for unknown value")
	}
}

func TestDescriptions(t *testing.T) {
	ctx := context.Background()
	validators := []validator.String{
		AccountName{},
		BucketName{},
		IAMUsername(),
		IAMPolicyName(),
		Email{},
		Hostname{},
		JSONDocument{},
	}

	for _, v := range validators {
		desc := v.Description(ctx)
		md := v.MarkdownDescription(ctx)
		if desc == "" {
			t.Errorf("%T has empty description", v)
		}
		if md == "" {
			t.Errorf("%T has empty markdown description", v)
		}
	}
}
