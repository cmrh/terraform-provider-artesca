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

	accountID, ok := s.ids[accountName]
	if !ok {
		acct, err := s.iam.GetAccountByName(ctx, token, accountName)
		if err != nil {
			return Credentials{}, fmt.Errorf("looking up account %q: %w", accountName, err)
		}
		if acct == nil || acct.ID == "" {
			return Credentials{}, fmt.Errorf("account %q not found", accountName)
		}
		accountID = acct.ID
		s.ids[accountName] = accountID
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

// Forget drops cached credentials and the cached account ID for accountName,
// e.g. after the account is deleted and may be recreated with a new ID.
func (s *AccountCredentialSource) Forget(accountName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.ids, accountName)
	delete(s.cache, accountName)
}
