package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccLocation_basic(t *testing.T) {
	rName := randomName("tf-acc-loc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLocationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccLocationSourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_location.source", "name", rName),
					resource.TestCheckResourceAttr("artesca_location.source", "location_type", "location-scality-ring-s3-v1"),
					resource.TestCheckResourceAttrSet("artesca_location.source", "object_id"),
					resource.TestCheckResourceAttr("artesca_location.source", "details.bucket_match", "false"),
					resource.TestCheckResourceAttr("artesca_location.source", "details.server_side_encryption", "true"),
				),
			},
		},
	})
}

func TestAccLocation_update(t *testing.T) {
	rName := randomName("tf-acc-loc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLocationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccLocationWithSSE(rName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_location.source", "details.server_side_encryption", "true"),
				),
			},
			{
				Config: testAccLocationWithSSE(rName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artesca_location.source", "details.server_side_encryption", "false"),
				),
			},
		},
	})
}

func TestAccLocation_importState(t *testing.T) {
	rName := randomName("tf-acc-loc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLocationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccLocationSourceConfig(rName),
			},
			{
				ResourceName:                         "artesca_location.source",
				ImportState:                          true,
				ImportStateId:                        rName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerifyIgnore:              []string{"details.secret_key", "details.access_key", "details.bucket_match", "details.bucket_name", "details.endpoint", "details.region"},
			},
		},
	})
}

func TestAccLocation_validateConfigSproxydMissingFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "artesca_location" "test" {
  name          = "tf-acc-loc-sproxyd"
  location_type = "location-scality-sproxyd-v1"

  details {
    bootstrap_list = ["10.0.0.1:8181"]
  }
}
`,
				ExpectError: regexp.MustCompile(`(?s)details\.chord_cos is required.*location-scality-sproxyd-v1`),
			},
		},
	})
}

func TestAccLocation_validateConfigAwsS3MissingDetails(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "artesca_location" "test" {
  name          = "tf-acc-loc-aws"
  location_type = "location-aws-s3-v1"
}
`,
				ExpectError: regexp.MustCompile(`(?s)Missing details block.*location-aws-s3-v1`),
			},
		},
	})
}

func TestAccLocation_validateConfigUnknownTypeSkipsValidation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "artesca_location" "test" {
  name          = "tf-acc-loc-mem"
  location_type = "location-mem-v1"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccLocationWithSSE(name string, sse bool) string {
	return fmt.Sprintf(`
resource "artesca_location" "source" {
  name          = %q
  location_type = "location-scality-ring-s3-v1"

  details {
    endpoint               = "%s"
    access_key             = "%s"
    secret_key             = "%s"
    bucket_name            = "%s"
    bucket_match           = false
    server_side_encryption = %t
  }
}
`, name,
		os.Getenv("TF_VAR_ring_s3_endpoint"),
		os.Getenv("TF_VAR_ring_s3_access_key"),
		os.Getenv("TF_VAR_ring_s3_secret_key"),
		os.Getenv("TF_VAR_ring_s3_bucket_name"),
		sse,
	)
}
