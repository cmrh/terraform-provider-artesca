package provider

import (
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Lifecycle validation fails at plan time; no resources are created.
func TestAccWorkflowLifecycle_validateConfig(t *testing.T) {
	cases := []struct {
		name, config, err string
	}{
		{"expiration days below 1", `
resource "artesca_bucket_workflow_expiration" "test" {
  account_name                       = "tf-acc-validate"
  bucket_name                        = "tf-acc-validate"
  enabled                            = true
  current_version_trigger_delay_days = 0
}
`, `(?s)Value too small.*at least 1, got 0`},
		{"transition days below 0", `
resource "artesca_bucket_workflow_transition" "test" {
  account_name       = "tf-acc-validate"
  bucket_name        = "tf-acc-validate"
  enabled            = true
  location_name      = "tf-acc-validate"
  trigger_delay_days = -1
}
`, `(?s)Value too small.*at least 0, got -1`},
		{"rule_id longer than 255", `
resource "artesca_bucket_workflow_expiration" "test" {
  account_name                       = "tf-acc-validate"
  bucket_name                        = "tf-acc-validate"
  rule_id                            = "` + strings.Repeat("r", 256) + `"
  enabled                            = true
  current_version_trigger_delay_days = 30
}
`, `(?s)Invalid rule ID.*1–255 characters, got 256`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config:      tc.config,
					ExpectError: regexp.MustCompile(tc.err),
				}},
			})
		})
	}
}
