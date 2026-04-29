package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"artesca": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	for _, env := range []string{
		"ARTESCA_MANAGEMENT_ENDPOINT",
		"ARTESCA_OIDC_URL",
		"ARTESCA_USERNAME",
		"ARTESCA_PASSWORD",
	} {
		if os.Getenv(env) == "" {
			t.Fatalf("%s must be set for acceptance tests", env)
		}
	}
}

func testAccPreCheckS3(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)
	if os.Getenv("ARTESCA_S3_ENDPOINT") == "" {
		t.Fatal("ARTESCA_S3_ENDPOINT must be set for bucket tests")
	}
}

func testAccPreCheckRingS3(t *testing.T) {
	t.Helper()
	testAccPreCheckS3(t)
	for _, env := range []string{
		"TF_VAR_ring_s3_endpoint",
		"TF_VAR_ring_s3_access_key",
		"TF_VAR_ring_s3_secret_key",
		"TF_VAR_ring_s3_bucket_name",
	} {
		if os.Getenv(env) == "" {
			t.Fatalf("%s must be set for location tests", env)
		}
	}
}

func testAccPreCheckDestRingS3(t *testing.T) {
	t.Helper()
	testAccPreCheckRingS3(t)
	for _, env := range []string{
		"TF_VAR_dest_ring_s3_endpoint",
		"TF_VAR_dest_ring_s3_access_key",
		"TF_VAR_dest_ring_s3_secret_key",
		"TF_VAR_dest_ring_s3_bucket_name",
	} {
		if os.Getenv(env) == "" {
			t.Fatalf("%s must be set for destination location tests", env)
		}
	}
}

func randomName(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
}

func testAccAccountConfig(name string) string {
	return fmt.Sprintf(`
resource "artesca_account" "test" {
  name  = %q
  email = "%s@test.example.com"
}
`, name, name)
}

func testAccLocationSourceConfig(name string) string {
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
    server_side_encryption = true
  }
}
`, name,
		os.Getenv("TF_VAR_ring_s3_endpoint"),
		os.Getenv("TF_VAR_ring_s3_access_key"),
		os.Getenv("TF_VAR_ring_s3_secret_key"),
		os.Getenv("TF_VAR_ring_s3_bucket_name"),
	)
}

func testAccLocationDestConfig(name string) string {
	return fmt.Sprintf(`
resource "artesca_location" "dest" {
  name          = %q
  location_type = "location-scality-ring-s3-v1"

  details {
    endpoint               = "%s"
    access_key             = "%s"
    secret_key             = "%s"
    bucket_name            = "%s"
    bucket_match           = false
    server_side_encryption = true
  }
}
`, name,
		os.Getenv("TF_VAR_dest_ring_s3_endpoint"),
		os.Getenv("TF_VAR_dest_ring_s3_access_key"),
		os.Getenv("TF_VAR_dest_ring_s3_secret_key"),
		os.Getenv("TF_VAR_dest_ring_s3_bucket_name"),
	)
}

func testAccBucketConfig(resourceName, bucketName, locationRef string, versioned bool) string {
	return fmt.Sprintf(`
resource "artesca_bucket" %q {
  name                = %q
  location_constraint = %s
  versioning_enabled  = %t
  account_access_key  = artesca_account.test.access_key
  account_secret_key  = artesca_account.test.secret_key
}
`, resourceName, bucketName, locationRef, versioned)
}
