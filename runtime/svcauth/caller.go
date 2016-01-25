package svcauth

import "golang.org/x/net/context"

type AuthInfo struct {
	UserID   string
	ClientID string
	Operator bool

	// User or Repo id
	SubjectType string
	SubjectID   string
}

type Authenticator interface {
	Authn(ctx context.Context, rule *AuthnRule, info *AuthInfo) error
	Authz(ctx context.Context, rule *AuthzRule, info *AuthInfo) error
}

type authInfoKey struct{}

func ContextWithAuthInfo(ctx context.Context, info *AuthInfo) context.Context {
	return context.WithValue(ctx, authInfoKey{}, info)
}

func Auth(ctx context.Context) *AuthInfo {
	v, _ := ctx.Value(authInfoKey{}).(*AuthInfo)
	return v
}
