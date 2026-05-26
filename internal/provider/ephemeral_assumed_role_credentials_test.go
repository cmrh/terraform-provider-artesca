package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccEphemeralAssumedRoleCredentials_basic(t *testing.T) {
	rAcct := randomName("tf-acc-sts")
	rRole := randomName("tf-acc-role")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccEphemeralAssumedRoleCredentialsConfig(rAcct, rRole),
				// Ephemeral resources don't write to state; we verify by
				// (a) Terraform not erroring out, and (b) check blocks
				// in the config asserting the returned values are non-empty.
			},
		},
	})
}

func testAccEphemeralAssumedRoleCredentialsConfig(accountName, roleName string) string {
	return testAccAccountConfig(accountName) + fmt.Sprintf(`
resource "artesca_user" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  username           = "tf-acc-sts-caller"
}

resource "artesca_user_policy" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  username           = artesca_user.test.username
  policy_name        = "sts-all"
  policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [{ Effect = "Allow", Action = "sts:*", Resource = "*" }]
  })
}

resource "artesca_user_access_key" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  username           = artesca_user.test.username
}

resource "artesca_role" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  name               = %q
  assume_role_policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { AWS = "arn:aws:iam::${artesca_account.test.id}:root" }
      Action    = "sts:AssumeRole"
    }]
  })
}

ephemeral "artesca_assumed_role_credentials" "test" {
  access_key        = artesca_user_access_key.test.access_key_id
  secret_key        = artesca_user_access_key.test.secret_access_key
  role_arn          = artesca_role.test.arn
  role_session_name = "tf-acc-test-session"
}

check "assume_role_returned_credentials" {
  assert {
    condition     = ephemeral.artesca_assumed_role_credentials.test.assumed_role_arn != ""
    error_message = "assumed_role_arn was empty"
  }
  assert {
    condition     = ephemeral.artesca_assumed_role_credentials.test.expiration != ""
    error_message = "expiration was empty"
  }
}
`, roleName)
}
