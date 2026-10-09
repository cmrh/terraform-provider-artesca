package provider

import (
	"context"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Policy documents changed outside Terraform must show up in the next plan.

const testAccDriftDocument = `{"Version":"2012-10-17","Statement":[{"Effect":"Deny","Action":"s3:DeleteObject","Resource":"*"}]}`

// testAccOutOfBand runs change with an IAM client and the account's credentials.
func testAccOutOfBand(t *testing.T, accountName string, change func(context.Context, *client.IAMClient, client.Credentials) error) func() {
	return func() {
		rs := &terraform.ResourceState{Primary: &terraform.InstanceState{Attributes: map[string]string{"account_name": accountName}}}
		creds, ok := testAccAccountCredentials(rs)
		if !ok {
			t.Fatalf("no credentials for account %s", accountName)
		}
		iam, err := testAccIAMClient()
		if err != nil {
			t.Fatal(err)
		}
		if err := change(context.Background(), iam, creds); err != nil {
			t.Fatalf("changing the document outside Terraform: %v", err)
		}
	}
}

func TestAccUserPolicy_documentDrift(t *testing.T) {
	rAcct := randomName("tf-acc-up")
	rUser := randomName("tf-acc-user")
	config := testAccUserPolicyConfig(rAcct, rUser, "s3:GetObject")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: testAccOutOfBand(t, rAcct, func(ctx context.Context, iam *client.IAMClient, c client.Credentials) error {
					return iam.PutUserPolicy(ctx, c, rUser, "tf-acc-policy", testAccDriftDocument)
				}),
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccGroupPolicy_documentDrift(t *testing.T) {
	rAcct := randomName("tf-acc-gp")
	rGroup := randomName("tf-acc-group")
	rPolicy := randomName("tf-acc-pol")
	config := testAccGroupPolicyConfig(rAcct, rGroup, rPolicy)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: testAccOutOfBand(t, rAcct, func(ctx context.Context, iam *client.IAMClient, c client.Credentials) error {
					return iam.PutGroupPolicy(ctx, c, rGroup, rPolicy, testAccDriftDocument)
				}),
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccPolicy_documentDrift(t *testing.T) {
	rAcct := randomName("tf-acc-mp")
	rPolicy := randomName("tf-acc-pol")
	config := testAccPolicyConfig(rAcct, rPolicy)
	var arn string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: func(s *terraform.State) error {
					arn = s.RootModule().Resources["artesca_policy.test"].Primary.Attributes["arn"]
					return nil
				},
			},
			{
				PreConfig: testAccOutOfBand(t, rAcct, func(ctx context.Context, iam *client.IAMClient, c client.Credentials) error {
					if err := iam.CreatePolicyVersion(ctx, c, arn, testAccDriftDocument); err != nil {
						return err
					}
					// v1 is no longer the default; remove it so the policy can be deleted on destroy.
					return iam.DeletePolicyVersion(ctx, c, arn, "v1")
				}),
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccRole_trustPolicyDrift(t *testing.T) {
	rAcct := randomName("tf-acc-role")
	rRole := randomName("tf-acc-role")
	config := testAccRoleConfig(rAcct, rRole)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				// ARTESCA can't update a trust policy, so recreate the role with a different one.
				PreConfig: testAccOutOfBand(t, rAcct, func(ctx context.Context, iam *client.IAMClient, c client.Credentials) error {
					if err := iam.DeleteRole(ctx, c, rRole); err != nil {
						return err
					}
					_, err := iam.CreateRole(ctx, c, rRole,
						`{"Version":"2012-10-17","Statement":[{"Effect":"Deny","Principal":{"AWS":"*"},"Action":"sts:AssumeRole"}]}`, "")
					return err
				}),
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
