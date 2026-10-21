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

// Objects added outside Terraform block a destroy with a clear error;
// force_destroy (user, bucket) removes them.

func testAccOutOfBandS3(t *testing.T, accountName string, change func(context.Context, *client.S3Client, client.Credentials) error) func() {
	return func() {
		rs := &terraform.ResourceState{Primary: &terraform.InstanceState{Attributes: map[string]string{"account_name": accountName}}}
		creds, ok := testAccAccountCredentials(rs)
		if !ok {
			t.Fatalf("no credentials for account %s", accountName)
		}
		if err := change(context.Background(), testAccS3Client(), creds); err != nil {
			t.Fatalf("changing the bucket outside Terraform: %v", err)
		}
	}
}

func TestAccBucket_forceDestroy(t *testing.T) {
	rAcct, rBucket := randomName("tf-acc"), randomName("tf-acc-bkt")
	config := func(force bool) string {
		return testAccAccountConfig(rAcct) + fmt.Sprintf(`
resource "artesca_bucket" "test" {
  name               = %q
  account_name       = artesca_account.test.name
  versioning_enabled = true
  force_destroy      = %t
}
`, rBucket, force)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBucketDestroy,
		Steps: []resource.TestStep{
			{Config: config(false)},
			{
				PreConfig: testAccOutOfBandS3(t, rAcct, func(ctx context.Context, s3 *client.S3Client, c client.Credentials) error {
					for _, key := range []string{"a", "dir/b c", "a"} {
						if err := s3.PutObject(ctx, c, rBucket, key, "data"); err != nil {
							return err
						}
					}
					return s3.DeleteObject(ctx, c, rBucket, "a") // delete marker
				}),
				Config:      config(false),
				Destroy:     true,
				ExpectError: regexp.MustCompile(`Bucket not empty`),
			},
			{Config: config(true)},
		},
	})
}

func TestAccUser_forceDestroy(t *testing.T) {
	rAcct, rUser, rGroup, rPolicy := randomName("tf-acc"), randomName("tf-acc-user"), randomName("tf-acc-group"), randomName("tf-acc-pol")
	config := func(force bool) string {
		return testAccAccountConfig(rAcct) + fmt.Sprintf(`
# The user depends on the group and policy, so it is destroyed first and
# leaves them before they are deleted.
resource "artesca_user" "test" {
  account_name  = artesca_account.test.name
  username      = %q
  force_destroy = %t
  depends_on    = [artesca_group.test, artesca_policy.test]
}

resource "artesca_group" "test" {
  account_name = artesca_account.test.name
  name         = %q
}

resource "artesca_policy" "test" {
  account_name    = artesca_account.test.name
  name            = %q
  policy_document = jsonencode({ Version = "2012-10-17", Statement = [{ Effect = "Allow", Action = "s3:GetObject", Resource = "*" }] })
}
`, rUser, force, rGroup, rPolicy)
	}
	var policyArn string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: config(false),
				Check: func(s *terraform.State) error {
					policyArn = s.RootModule().Resources["artesca_policy.test"].Primary.Attributes["arn"]
					return nil
				},
			},
			{
				PreConfig: testAccOutOfBand(t, rAcct, func(ctx context.Context, iam *client.IAMClient, c client.Credentials) error {
					if err := iam.AttachUserPolicy(ctx, c, rUser, policyArn); err != nil {
						return err
					}
					return iam.AddUserToGroup(ctx, c, rGroup, rUser)
				}),
				Config:      config(false),
				Destroy:     true,
				ExpectError: regexp.MustCompile(`User has dependents`),
			},
			{Config: config(true)},
		},
	})
}

func TestAccGroup_deleteWithMember(t *testing.T) {
	rAcct, rGroup := randomName("tf-acc"), randomName("tf-acc-group")
	config := testAccGroupConfig(rAcct, rGroup)
	const member = "tf-acc-member"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: testAccOutOfBand(t, rAcct, func(ctx context.Context, iam *client.IAMClient, c client.Credentials) error {
					if _, err := iam.CreateUser(ctx, c, member); err != nil {
						return err
					}
					return iam.AddUserToGroup(ctx, c, rGroup, member)
				}),
				Config:      config,
				Destroy:     true,
				ExpectError: regexp.MustCompile(`Group has dependents`),
			},
			{
				PreConfig: testAccOutOfBand(t, rAcct, func(ctx context.Context, iam *client.IAMClient, c client.Credentials) error {
					if err := iam.RemoveUserFromGroup(ctx, c, rGroup, member); err != nil {
						return err
					}
					return iam.DeleteUser(ctx, c, member)
				}),
				Config: config,
			},
		},
	})
}

func TestAccRole_deleteWithAttachedPolicy(t *testing.T) {
	rAcct, rRole := randomName("tf-acc"), randomName("tf-acc-role")
	config := testAccRoleConfig(rAcct, rRole)
	var policyArn string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: testAccOutOfBand(t, rAcct, func(ctx context.Context, iam *client.IAMClient, c client.Credentials) error {
					p, err := iam.CreatePolicy(ctx, c, "tf-acc-oob", `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:GetObject","Resource":"*"}]}`, "")
					if err != nil {
						return err
					}
					policyArn = p.Arn
					return iam.AttachRolePolicy(ctx, c, rRole, policyArn)
				}),
				Config:      config,
				Destroy:     true,
				ExpectError: regexp.MustCompile(`Role has dependents`),
			},
			{
				PreConfig: testAccOutOfBand(t, rAcct, func(ctx context.Context, iam *client.IAMClient, c client.Credentials) error {
					if err := iam.DetachRolePolicy(ctx, c, rRole, policyArn); err != nil {
						return err
					}
					return iam.DeletePolicy(ctx, c, policyArn)
				}),
				Config: config,
			},
		},
	})
}

func TestAccAccount_deleteNotEmpty(t *testing.T) {
	rAcct := randomName("tf-acc")
	config := testAccAccountConfig(rAcct)
	bucket := rAcct + "-oob"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAccountDestroy,
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: testAccOutOfBandS3(t, rAcct, func(ctx context.Context, s3 *client.S3Client, c client.Credentials) error {
					return s3.CreateBucket(ctx, c, bucket, "", false)
				}),
				Config:      config,
				Destroy:     true,
				ExpectError: regexp.MustCompile(`Account not empty`),
			},
			{
				PreConfig: testAccOutOfBandS3(t, rAcct, func(ctx context.Context, s3 *client.S3Client, c client.Credentials) error {
					return s3.DeleteBucket(ctx, c, bucket)
				}),
				Config: config,
			},
		},
	})
}
