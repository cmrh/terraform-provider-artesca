package client

// ProviderClients bundles all API clients, passed via resp.ResourceData.
type ProviderClients struct {
	Management *ManagementClient
	IAM        *IAMClient
	S3         *S3Client
}
