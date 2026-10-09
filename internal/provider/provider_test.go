package provider

import (
	"context"
	"fmt"
	"os"
	"slices"
	"sync"
	"testing"

	"github.com/cmrh/terraform-provider-artesca/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
    endpoint     = "%s"
    access_key   = "%s"
    secret_key   = "%s"
    bucket_name  = "%s"
    bucket_match = false
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
    endpoint     = "%s"
    access_key   = "%s"
    secret_key   = "%s"
    bucket_name  = "%s"
    bucket_match = false
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
  account_name = artesca_account.test.name
}
`, resourceName, bucketName, locationRef, versioned)
}

func testAccManagementClient() (*client.ManagementClient, error) {
	endpoint := os.Getenv("ARTESCA_MANAGEMENT_ENDPOINT")
	oidcURL := os.Getenv("ARTESCA_OIDC_URL")
	username := os.Getenv("ARTESCA_USERNAME")
	password := os.Getenv("ARTESCA_PASSWORD")

	realm := os.Getenv("ARTESCA_OIDC_REALM")
	if realm == "" {
		realm = "artesca"
	}
	clientID := os.Getenv("ARTESCA_CLIENT_ID")
	if clientID == "" {
		clientID = "zenko-ui"
	}
	scope := os.Getenv("ARTESCA_OIDC_SCOPE")
	if scope == "" {
		scope = "openid"
	}
	insecure := testAccInsecure()

	tokenSource := client.NewOIDCTokenSource(oidcURL, realm, clientID, scope, username, password, insecure)
	mgmtClient := client.NewManagementClient(endpoint, "", tokenSource, insecure)

	instanceID := os.Getenv("ARTESCA_INSTANCE_ID")
	if instanceID == "" {
		ids, err := tokenSource.InstanceIDs(context.Background())
		if err != nil {
			return nil, err
		}
		if len(ids) > 0 {
			instanceID = ids[0]
		}
	}
	mgmtClient.InstanceID = instanceID

	return mgmtClient, nil
}

// testAccInsecure reports whether TLS verification is disabled for the test cluster.
func testAccInsecure() bool {
	v := os.Getenv("ARTESCA_INSECURE_SKIP_VERIFY")
	return v == "true" || v == "1"
}

// testAccS3Client returns an S3 client for ARTESCA_S3_ENDPOINT.
func testAccS3Client() *client.S3Client {
	return client.NewS3Client(os.Getenv("ARTESCA_S3_ENDPOINT"), "us-east-1", testAccInsecure())
}

// testAccIAMClient returns an IAM client for the endpoint the provider derives
// from ARTESCA_MANAGEMENT_ENDPOINT.
func testAccIAMClient() (*client.IAMClient, error) {
	iamEndpoint, err := client.DeriveIAMEndpoint(os.Getenv("ARTESCA_MANAGEMENT_ENDPOINT"))
	if err != nil {
		return nil, err
	}
	region := os.Getenv("ARTESCA_IAM_REGION")
	if region == "" {
		region = "us-east-1"
	}
	insecure := testAccInsecure()
	return client.NewIAMClient(iamEndpoint, region, insecure), nil
}

var (
	testAccAccountsOnce sync.Once
	testAccAccounts     *client.AccountCredentialSource
	testAccAccountsErr  error
)

// testAccAccountCredentials returns credentials for the account that owns an
// account-scoped resource, the same way the provider obtains them. ok is false
// when credentials can't be obtained (e.g. the account was already deleted);
// destroy checks treat that as destroyed.
func testAccAccountCredentials(rs *terraform.ResourceState) (client.Credentials, bool) {
	testAccAccountsOnce.Do(func() {
		mgmtClient, err := testAccManagementClient()
		if err != nil {
			testAccAccountsErr = err
			return
		}
		iamClient, err := testAccIAMClient()
		if err != nil {
			testAccAccountsErr = err
			return
		}
		var stsEndpoint string
		if s3 := os.Getenv("ARTESCA_S3_ENDPOINT"); s3 != "" {
			stsEndpoint, err = client.DeriveSTSEndpoint(s3)
		} else {
			stsEndpoint, err = client.DeriveSTSEndpointFromManagement(os.Getenv("ARTESCA_MANAGEMENT_ENDPOINT"))
		}
		if err != nil {
			testAccAccountsErr = err
			return
		}
		insecure := testAccInsecure()
		testAccAccounts = client.NewAccountCredentialSource(iamClient, client.NewSTSClient(stsEndpoint, "us-east-1", insecure), mgmtClient.TokenSource)
	})
	if testAccAccountsErr != nil {
		return client.Credentials{}, false
	}
	accountName := rs.Primary.Attributes["account_name"]
	// Destroy checks run after the account may have been deleted and recreated.
	testAccAccounts.Forget(accountName)
	c, err := testAccAccounts.For(context.Background(), accountName)
	if err != nil {
		return client.Credentials{}, false
	}
	return c, true
}

func testAccCheckAccountDestroy(s *terraform.State) error {
	mgmtClient, err := testAccManagementClient()
	if err != nil {
		return err
	}
	iamClient, err := testAccIAMClient()
	if err != nil {
		return err
	}
	token, err := mgmtClient.TokenSource.Token(context.Background())
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_account" {
			continue
		}
		acct, err := iamClient.GetAccountByName(context.Background(), token, rs.Primary.Attributes["name"])
		if err != nil {
			return err
		}
		if acct != nil {
			return fmt.Errorf("account %s still exists", rs.Primary.Attributes["name"])
		}
	}
	return nil
}

func testAccCheckLocationDestroy(s *terraform.State) error {
	mgmtClient, err := testAccManagementClient()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_location" {
			continue
		}
		loc, err := mgmtClient.GetLocation(context.Background(), rs.Primary.Attributes["name"])
		if err != nil {
			return err
		}
		if loc != nil {
			return fmt.Errorf("location %s still exists", rs.Primary.Attributes["name"])
		}
	}
	return nil
}

func testAccCheckEndpointDestroy(s *terraform.State) error {
	mgmtClient, err := testAccManagementClient()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_endpoint" {
			continue
		}
		ep, err := mgmtClient.GetEndpoint(context.Background(), rs.Primary.Attributes["hostname"])
		if err != nil {
			return err
		}
		if ep != nil {
			return fmt.Errorf("endpoint %s still exists", rs.Primary.Attributes["hostname"])
		}
	}
	return nil
}

func testAccCheckReplicationDestroy(s *terraform.State) error {
	mgmtClient, err := testAccManagementClient()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_replication" {
			continue
		}
		stream, err := mgmtClient.GetReplicationStream(context.Background(), rs.Primary.Attributes["stream_id"])
		if err != nil {
			return err
		}
		if stream != nil {
			return fmt.Errorf("replication stream %s still exists", rs.Primary.Attributes["stream_id"])
		}
	}
	return nil
}

func testAccCheckBucketDestroy(s *terraform.State) error {
	s3Endpoint := os.Getenv("ARTESCA_S3_ENDPOINT")
	if s3Endpoint == "" {
		return nil
	}
	s3Client := testAccS3Client()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_bucket" {
			continue
		}
		acctCreds, ok := testAccAccountCredentials(rs)
		if !ok {
			continue
		}
		exists, err := s3Client.HeadBucket(context.Background(),
			acctCreds,
			rs.Primary.Attributes["name"])
		if err != nil {
			continue
		}
		if exists {
			return fmt.Errorf("bucket %s still exists", rs.Primary.Attributes["name"])
		}
	}
	return nil
}

func testAccCheckUserDestroy(s *terraform.State) error {
	iamClient, err := testAccIAMClient()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_user" {
			continue
		}
		acctCreds, ok := testAccAccountCredentials(rs)
		if !ok {
			continue
		}
		user, err := iamClient.GetUser(context.Background(),
			acctCreds,
			rs.Primary.Attributes["username"])
		if err != nil {
			continue
		}
		if user != nil {
			return fmt.Errorf("user %s still exists", rs.Primary.Attributes["username"])
		}
	}
	return nil
}

func testAccCheckUserAccessKeyDestroy(s *terraform.State) error {
	iamClient, err := testAccIAMClient()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_user_access_key" {
			continue
		}
		acctCreds, ok := testAccAccountCredentials(rs)
		if !ok {
			continue
		}
		keys, err := iamClient.ListAccessKeys(context.Background(),
			acctCreds,
			rs.Primary.Attributes["username"])
		if err != nil {
			continue
		}
		keyID := rs.Primary.Attributes["access_key_id"]
		for _, k := range keys {
			if k.AccessKeyId == keyID {
				return fmt.Errorf("access key %s still exists", keyID)
			}
		}
	}
	return nil
}

func testAccCheckUserPolicyDestroy(s *terraform.State) error {
	iamClient, err := testAccIAMClient()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_user_policy" {
			continue
		}
		acctCreds, ok := testAccAccountCredentials(rs)
		if !ok {
			continue
		}
		doc, err := iamClient.GetUserPolicy(context.Background(),
			acctCreds,
			rs.Primary.Attributes["username"],
			rs.Primary.Attributes["policy_name"])
		if err != nil {
			continue
		}
		if doc != "" {
			return fmt.Errorf("user policy %s still exists for user %s", rs.Primary.Attributes["policy_name"], rs.Primary.Attributes["username"])
		}
	}
	return nil
}

func testAccCheckGroupDestroy(s *terraform.State) error {
	iamClient, err := testAccIAMClient()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_group" {
			continue
		}
		acctCreds, ok := testAccAccountCredentials(rs)
		if !ok {
			continue
		}
		g, err := iamClient.GetGroup(context.Background(),
			acctCreds,
			rs.Primary.Attributes["name"])
		if err != nil {
			continue
		}
		if g != nil {
			return fmt.Errorf("group %s still exists", rs.Primary.Attributes["name"])
		}
	}
	return nil
}

func testAccCheckGroupMembershipDestroy(s *terraform.State) error {
	iamClient, err := testAccIAMClient()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_group_membership" {
			continue
		}
		acctCreds, ok := testAccAccountCredentials(rs)
		if !ok {
			continue
		}
		groups, err := iamClient.ListGroupsForUser(context.Background(),
			acctCreds,
			rs.Primary.Attributes["username"])
		if err != nil {
			continue
		}
		want := rs.Primary.Attributes["group_name"]
		if slices.Contains(groups, want) {
			return fmt.Errorf("user %s still in group %s", rs.Primary.Attributes["username"], want)
		}
	}
	return nil
}

func testAccCheckGroupPolicyDestroy(s *terraform.State) error {
	iamClient, err := testAccIAMClient()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_group_policy" {
			continue
		}
		acctCreds, ok := testAccAccountCredentials(rs)
		if !ok {
			continue
		}
		doc, err := iamClient.GetGroupPolicy(context.Background(),
			acctCreds,
			rs.Primary.Attributes["group_name"],
			rs.Primary.Attributes["policy_name"])
		if err != nil {
			continue
		}
		if doc != "" {
			return fmt.Errorf("group policy %s still exists on %s", rs.Primary.Attributes["policy_name"], rs.Primary.Attributes["group_name"])
		}
	}
	return nil
}

func testAccCheckRoleDestroy(s *terraform.State) error {
	iamClient, err := testAccIAMClient()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_role" {
			continue
		}
		acctCreds, ok := testAccAccountCredentials(rs)
		if !ok {
			continue
		}
		role, err := iamClient.GetRole(context.Background(),
			acctCreds,
			rs.Primary.Attributes["name"])
		if err != nil {
			continue
		}
		if role != nil {
			return fmt.Errorf("role %s still exists", rs.Primary.Attributes["name"])
		}
	}
	return nil
}

func testAccCheckPolicyDestroy(s *terraform.State) error {
	iamClient, err := testAccIAMClient()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_policy" {
			continue
		}
		acctCreds, ok := testAccAccountCredentials(rs)
		if !ok {
			continue
		}
		pol, err := iamClient.GetPolicy(context.Background(),
			acctCreds,
			rs.Primary.Attributes["arn"])
		if err != nil {
			continue
		}
		if pol != nil {
			return fmt.Errorf("managed policy %s still exists", rs.Primary.Attributes["arn"])
		}
	}
	return nil
}

func testAccCheckUserPolicyAttachmentDestroy(s *terraform.State) error {
	iamClient, err := testAccIAMClient()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_user_policy_attachment" {
			continue
		}
		acctCreds, ok := testAccAccountCredentials(rs)
		if !ok {
			continue
		}
		arns, err := iamClient.ListAttachedUserPolicies(context.Background(),
			acctCreds,
			rs.Primary.Attributes["username"])
		if err != nil {
			continue
		}
		want := rs.Primary.Attributes["policy_arn"]
		if slices.Contains(arns, want) {
			return fmt.Errorf("user %s still has policy %s attached", rs.Primary.Attributes["username"], want)
		}
	}
	return nil
}

func testAccCheckGroupPolicyAttachmentDestroy(s *terraform.State) error {
	iamClient, err := testAccIAMClient()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_group_policy_attachment" {
			continue
		}
		acctCreds, ok := testAccAccountCredentials(rs)
		if !ok {
			continue
		}
		arns, err := iamClient.ListAttachedGroupPolicies(context.Background(),
			acctCreds,
			rs.Primary.Attributes["group_name"])
		if err != nil {
			continue
		}
		want := rs.Primary.Attributes["policy_arn"]
		if slices.Contains(arns, want) {
			return fmt.Errorf("group %s still has policy %s attached", rs.Primary.Attributes["group_name"], want)
		}
	}
	return nil
}

func testAccCheckRolePolicyAttachmentDestroy(s *terraform.State) error {
	iamClient, err := testAccIAMClient()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "artesca_role_policy_attachment" {
			continue
		}
		acctCreds, ok := testAccAccountCredentials(rs)
		if !ok {
			continue
		}
		arns, err := iamClient.ListAttachedRolePolicies(context.Background(),
			acctCreds,
			rs.Primary.Attributes["role_name"])
		if err != nil {
			continue
		}
		want := rs.Primary.Attributes["policy_arn"]
		if slices.Contains(arns, want) {
			return fmt.Errorf("role %s still has policy %s attached", rs.Primary.Attributes["role_name"], want)
		}
	}
	return nil
}

// testAccImportWithAccount wraps an ImportStateIdFunc for account-scoped
// resources, prefixing the ID with artesca_account.test's name to form
// "<account_name>/<id>".
func testAccImportWithAccount(idFunc resource.ImportStateIdFunc) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources["artesca_account.test"]
		if !ok {
			return "", fmt.Errorf("artesca_account.test not found in state")
		}
		accountName := rs.Primary.Attributes["name"]
		if accountName == "" {
			return "", fmt.Errorf("artesca_account.test has no name in state")
		}
		id, err := idFunc(s)
		if err != nil {
			return "", err
		}
		return accountName + "/" + id, nil
	}
}

// testAccImportStateID returns an ImportStateIdFunc for a fixed import ID.
func testAccImportStateID(id string) resource.ImportStateIdFunc {
	return func(*terraform.State) (string, error) {
		return id, nil
	}
}
