package client

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	// accountRoleName is the role ARTESCA provisions in every account for
	// storage management; the UI assumes the same role.
	accountRoleName           = "scality-internal/storage-manager-role"
	accountRoleSessionName    = "terraform-provider-artesca"
	accountCredentialDuration = 3600 * time.Second
	accountCredentialRefresh  = 5 * time.Minute
)

// AccountCredentialSource issues temporary credentials for an account by name:
// it resolves the account ID via IAM GetRolesForWebIdentity, then calls STS
// AssumeRoleWithWebIdentity on the account's storage-manager role with the
// provider's OIDC token. Credentials are cached per account until shortly
// before they expire.
type AccountCredentialSource struct {
	iam    *IAMClient
	sts    *STSClient
	tokens *OIDCTokenSource
	now    func() time.Time

	mu    sync.Mutex
	ids   map[string]string
	cache map[string]cachedAccountCredentials
}

type cachedAccountCredentials struct {
	creds   Credentials
	expires time.Time
}

func NewAccountCredentialSource(iam *IAMClient, sts *STSClient, tokens *OIDCTokenSource) *AccountCredentialSource {
	return &AccountCredentialSource{
		iam:    iam,
		sts:    sts,
		tokens: tokens,
		now:    time.Now,
		ids:    map[string]string{},
		cache:  map[string]cachedAccountCredentials{},
	}
}

// For returns credentials for the named account.
func (s *AccountCredentialSource) For(ctx context.Context, accountName string) (Credentials, error) {
	if accountName == "" {
		return Credentials{}, fmt.Errorf("account_name is empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if c, ok := s.cache[accountName]; ok && s.now().Before(c.expires.Add(-accountCredentialRefresh)) {
		return c.creds, nil
	}

	token, err := s.tokens.Token(ctx)
	if err != nil {
		return Credentials{}, fmt.Errorf("getting auth token: %w", err)
	}

	accountID, err := s.lookupID(ctx, token, accountName)
	if err != nil {
		return Credentials{}, err
	}

	roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, accountRoleName)
	assumed, err := s.sts.AssumeRoleWithWebIdentity(ctx, token, roleArn, accountRoleSessionName, int64(accountCredentialDuration/time.Second))
	if err != nil {
		return Credentials{}, fmt.Errorf("getting credentials for account %q: %w", accountName, err)
	}

	creds := Credentials{
		AccessKey:    assumed.AccessKeyID,
		SecretKey:    assumed.SecretAccessKey,
		SessionToken: assumed.SessionToken,
	}
	s.cache[accountName] = cachedAccountCredentials{creds: creds, expires: assumed.Expiration}
	return creds, nil
}

// AccountID returns the ID of the named account.
func (s *AccountCredentialSource) AccountID(ctx context.Context, accountName string) (string, error) {
	if accountName == "" {
		return "", fmt.Errorf("account_name is empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	token, err := s.tokens.Token(ctx)
	if err != nil {
		return "", fmt.Errorf("getting auth token: %w", err)
	}
	return s.lookupID(ctx, token, accountName)
}

// lookupID returns the account's ID from the cache or IAM. s.mu must be held.
func (s *AccountCredentialSource) lookupID(ctx context.Context, token, accountName string) (string, error) {
	if id, ok := s.ids[accountName]; ok {
		return id, nil
	}
	acct, err := s.iam.GetAccountByName(ctx, token, accountName)
	if err != nil {
		return "", fmt.Errorf("looking up account %q: %w", accountName, err)
	}
	if acct == nil || acct.ID == "" {
		return "", fmt.Errorf("account %q not found", accountName)
	}
	s.ids[accountName] = acct.ID
	return acct.ID, nil
}

// Forget drops cached credentials and the cached account ID for accountName,
// e.g. after the account is deleted and may be recreated with a new ID.
func (s *AccountCredentialSource) Forget(accountName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.ids, accountName)
	delete(s.cache, accountName)
}

// NameForAccountID returns the name of the account with the given ID.
func (s *AccountCredentialSource) NameForAccountID(ctx context.Context, accountID string) (string, error) {
	token, err := s.tokens.Token(ctx)
	if err != nil {
		return "", fmt.Errorf("getting auth token: %w", err)
	}
	accounts, err := s.iam.ListAccounts(ctx, token)
	if err != nil {
		return "", fmt.Errorf("listing accounts: %w", err)
	}
	for _, a := range accounts {
		if a.ID == accountID {
			return a.Name, nil
		}
	}
	return "", fmt.Errorf("no account with ID %q", accountID)
}

// NameForAccessKey returns the name of the account that owns an access key,
// via STS GetCallerIdentity.
func (s *AccountCredentialSource) NameForAccessKey(ctx context.Context, accessKey, secretKey string) (string, error) {
	identity, err := s.sts.GetCallerIdentity(ctx, accessKey, secretKey, "")
	if err != nil {
		return "", fmt.Errorf("identifying account for access key: %w", err)
	}
	return s.NameForAccountID(ctx, identity.Account)
}
