package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Empty values that mean nothing are rejected at validation, instead of
// failing after apply because ARTESCA drops or replaces them.
func TestAccEmptyValuesRejected(t *testing.T) {
	steps := []resource.TestStep{}
	for _, tc := range []struct {
		config string
		want   string
	}{
		{`
resource "artesca_location" "test" {
  name          = "tf-acc-unused"
  location_type = "location-scality-ring-s3-v1"
  details {
    endpoint    = "http://ring.example.com"
    access_key  = "ak"
    secret_key  = "sk"
    bucket_name = "unused"
    repo_id     = []
  }
}
`, `Empty list`},
		{`
resource "artesca_location" "test" {
  name          = "tf-acc-unused"
  location_type = "location-scality-ring-s3-v1"
  details {
    endpoint    = "http://ring.example.com"
    access_key  = "ak"
    secret_key  = "sk"
    bucket_name = "unused"
    region      = ""
  }
}
`, `Empty value`},
		{`
resource "artesca_replication" "test" {
  name    = "tf-acc-unused"
  enabled = true
  source {
    bucket_name = "unused"
    prefix      = ""
  }
  destination {
    role = ""
    locations {
      name = "unused"
    }
  }
}
`, `Empty value`},
		{`
resource "artesca_role" "test" {
  account_name                = "tf-acc-unused"
  name                        = "unused"
  description                 = ""
  assume_role_policy_document = jsonencode({ Version = "2012-10-17", Statement = [] })
}
`, `Empty value`},
		{`
resource "artesca_policy" "test" {
  account_name    = "tf-acc-unused"
  name            = "unused"
  description     = ""
  policy_document = jsonencode({ Version = "2012-10-17", Statement = [] })
}
`, `Empty value`},
	} {
		steps = append(steps, resource.TestStep{Config: tc.config, ExpectError: regexp.MustCompile(tc.want)})
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    steps,
	})
}
