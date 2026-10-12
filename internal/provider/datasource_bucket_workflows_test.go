package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceBucketWorkflows_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rSrcBkt := randomName("tf-acc-sbkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkflowReplicationConfig(rAcct, rLoc, rSrcBkt, true, "") + `
data "artesca_bucket_workflows" "lookup" {
  account_name = artesca_account.test.name
  bucket_name  = artesca_bucket.source.name
  depends_on   = [artesca_bucket_workflow_replication.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.artesca_bucket_workflows.lookup", "replications.#", "1"),
					resource.TestCheckResourceAttr("data.artesca_bucket_workflows.lookup", "replications.0.enabled", "true"),
					resource.TestCheckResourceAttr("data.artesca_bucket_workflows.lookup", "replications.0.source_bucket_name", rSrcBkt),
					resource.TestCheckResourceAttr("data.artesca_bucket_workflows.lookup", "replications.0.destination_location", rLoc),
					resource.TestCheckResourceAttr("data.artesca_bucket_workflows.lookup", "replications.0.destination_bucket_name", ""),
					resource.TestCheckResourceAttrSet("data.artesca_bucket_workflows.lookup", "replications.0.workflow_id"),
				),
			},
		},
	})
}

func TestAccDataSourceBucketWorkflows_empty(t *testing.T) {
	rAcct := randomName("tf-acc")
	rLoc := randomName("tf-acc-loc")
	rBkt := randomName("tf-acc-bkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(rAcct) +
					testAccLocationSourceConfig(rLoc) +
					testAccBucketConfig("test", rBkt, "artesca_location.source.name", false) + `
data "artesca_bucket_workflows" "lookup" {
  account_name = artesca_account.test.name
  bucket_name  = artesca_bucket.test.name
  depends_on   = [artesca_bucket.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.artesca_bucket_workflows.lookup", "replications.#", "0"),
					resource.TestCheckResourceAttr("data.artesca_bucket_workflows.lookup", "expirations.#", "0"),
					resource.TestCheckResourceAttr("data.artesca_bucket_workflows.lookup", "transitions.#", "0"),
				),
			},
		},
	})
}
