package client

// ProviderClients bundles all API clients, passed via resp.ResourceData.
type ProviderClients struct {
	Management *ManagementClient
	IAM        *IAMClient
	S3         *S3Client
	STS        *STSClient
	// Accounts issues per-account credentials for IAM and S3 calls.
	Accounts *AccountCredentialSource
}
