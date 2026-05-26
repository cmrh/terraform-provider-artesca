package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceBucketWorkflows_basic(t *testing.T) {
	rAcct := randomName("tf-acc")
	rSrcLoc := randomName("tf-acc-sloc")
	rDstLoc := randomName("tf-acc-dloc")
	rSrcBkt := randomName("tf-acc-sbkt")
	rDstBkt := randomName("tf-acc-dbkt")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDestRingS3(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkflowReplicationConfig(rAcct, rSrcLoc, rDstLoc, rSrcBkt, rDstBkt, 1, true) + `
data "artesca_bucket_workflows" "lookup" {
  account_id  = artesca_account.test.id
  bucket_name = artesca_bucket.source.name
  depends_on  = [artesca_bucket_workflow_replication.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.artesca_bucket_workflows.lookup", "replications.#", "1"),
					resource.TestCheckResourceAttr("data.artesca_bucket_workflows.lookup", "replications.0.enabled", "true"),
					resource.TestCheckResourceAttr("data.artesca_bucket_workflows.lookup", "replications.0.source_bucket_name", rSrcBkt),
					resource.TestCheckResourceAttr("data.artesca_bucket_workflows.lookup", "replications.0.destination_bucket_name", rDstBkt),
					resource.TestCheckResourceAttrSet("data.artesca_bucket_workflows.lookup", "replications.0.workflow_id"),
					resource.TestCheckResourceAttrSet("data.artesca_bucket_workflows.lookup", "instance_id"),
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
  account_id  = artesca_account.test.id
  bucket_name = artesca_bucket.test.name
  depends_on  = [artesca_bucket.test]
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
