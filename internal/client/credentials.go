package client

// Credentials signs IAM and S3 requests. SessionToken is set for temporary
// credentials (e.g. from AssumeRoleWithWebIdentity) and sent as the signed
// X-Amz-Security-Token header.
type Credentials struct {
	AccessKey    string
	SecretKey    string
	SessionToken string
}
