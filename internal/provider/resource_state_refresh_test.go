package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Objects deleted or replaced outside Terraform must leave state on refresh,
// so the next plan recreates them, rather than fail or be adopted.

func TestAccAccount_recreatedOutOfBand(t *testing.T) {
	rAcct := randomName("tf-acc")
	config := testAccAccountConfig(rAcct)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAccountDestroy,
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: func() {
					mgmt, err := testAccManagementClient()
					if err != nil {
						t.Fatal(err)
					}
					ctx := context.Background()
					if err := mgmt.DeleteAccount(ctx, rAcct); err != nil {
						t.Fatalf("deleting the account outside Terraform: %v", err)
					}
					if _, err := mgmt.CreateAccount(ctx, rAcct, rAcct+"@test.example.com"); err != nil {
						t.Fatalf("recreating the account outside Terraform: %v", err)
					}
				},
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccBucketSubresources_bucketDeletedOutOfBand(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rBucket := randomName("tf-acc-bkt")
	config := testAccAccountConfig(rAcct) +
		testAccLocationSourceConfig(rLoc) +
		testAccBucketConfig("test", rBucket, "null", true) +
		testAccBucketPolicyConfig(rBucket, "s3:GetObject") +
		testAccBucketTaggingConfig(map[string]string{"env": "test"}) +
		testAccBucketEncryptionConfig("AES256") + `
resource "artesca_bucket_workflow_expiration" "test" {
  account_name = artesca_account.test.name
  bucket_name  = artesca_bucket.test.name
  enabled      = true

  current_version_trigger_delay_days = 30
}

resource "artesca_bucket_workflow_replication" "test" {
  account_name         = artesca_account.test.name
  bucket_name          = artesca_bucket.test.name
  destination_location = artesca_location.source.name
  enabled              = true
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketDestroy,
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: func() {
					rs := &terraform.ResourceState{Primary: &terraform.InstanceState{Attributes: map[string]string{"account_name": rAcct}}}
					creds, ok := testAccAccountCredentials(rs)
					if !ok {
						t.Fatalf("no credentials for account %s", rAcct)
					}
					if err := testAccS3Client().DeleteBucket(context.Background(), creds, rBucket); err != nil {
						t.Fatalf("deleting the bucket outside Terraform: %v", err)
					}
				},
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccUserAccessKey_userDeletedOutOfBand(t *testing.T) {
	rAcct := randomName("tf-acc")
	rUser := randomName("tf-acc-user")
	config := testAccUserAccessKeyConfig(rAcct, rUser)

	var keyID string
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: func(s *terraform.State) error {
					keyID = s.RootModule().Resources["artesca_user_access_key.test"].Primary.Attributes["access_key_id"]
					return nil
				},
			},
			{
				PreConfig: testAccOutOfBand(t, rAcct, func(ctx context.Context, iam *client.IAMClient, c client.Credentials) error {
					if err := iam.DeleteAccessKey(ctx, c, rUser, keyID); err != nil {
						return err
					}
					return iam.DeleteUser(ctx, c, rUser)
				}),
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccWorkflowExpiration_duplicateRuleID(t *testing.T) {
	rAcct := randomName("tf-acc")
	rBucket := randomName("tf-acc-bkt")
	rule := func(name string) string {
		return fmt.Sprintf(`
resource "artesca_bucket_workflow_expiration" %q {
  account_name = artesca_account.test.name
  bucket_name  = artesca_bucket.test.name
  rule_id      = "tf-acc-dup"
  enabled      = true

  current_version_trigger_delay_days = 30
}
`, name)
	}
	config := testAccAccountConfig(rAcct) + testAccBucketConfig("test", rBucket, "null", false) + rule("first")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketDestroy,
		Steps: []resource.TestStep{
			{Config: config},
			{
				Config:      config + rule("second"),
				ExpectError: regexp.MustCompile(`(?s)Lifecycle rule already exists.*tf-acc-dup`),
			},
		},
	})
}

func TestAccBucketTagging_emptyTagsRejected(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "artesca_bucket_tagging" "test" {
  account_name = "tf-acc-unused"
  bucket_name  = "tf-acc-unused"
  tags         = {}
}
`,
				ExpectError: regexp.MustCompile(`Must have between 1 and 50 elements, got 0`),
			},
		},
	})
}
