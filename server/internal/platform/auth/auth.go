// Package auth はIDトークンの検証と、認証済みユーザーのコンテキスト受け渡しを行う。
package auth

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	firebase "firebase.google.com/go/v4"
	fbauth "firebase.google.com/go/v4/auth"
)

// Verifier はIDトークンを検証し、認証基盤上のUIDを返す。
type Verifier interface {
	Verify(ctx context.Context, token string) (authUID string, err error)
}

// DevVerifier は "dev:<uid>" 形式のトークンを受け付ける。ローカル開発専用。
type DevVerifier struct{}

func (DevVerifier) Verify(_ context.Context, token string) (string, error) {
	uid, ok := strings.CutPrefix(token, "dev:")
	if !ok || uid == "" {
		return "", errors.New("invalid dev token")
	}
	return "dev:" + uid, nil
}

// FirebaseVerifier は Firebase Authentication のIDトークンを検証する。
type FirebaseVerifier struct {
	client *fbauth.Client
}

func NewFirebaseVerifier(ctx context.Context, projectID string) (*FirebaseVerifier, error) {
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID})
	if err != nil {
		return nil, err
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, err
	}
	return &FirebaseVerifier{client: client}, nil
}

func (v *FirebaseVerifier) Verify(ctx context.Context, token string) (string, error) {
	t, err := v.client.VerifyIDToken(ctx, token)
	if err != nil {
		return "", err
	}
	return t.UID, nil
}

// UserResolver は認証基盤のUIDを内部ユーザーIDに解決する（初回は作成）。
type UserResolver interface {
	Resolve(ctx context.Context, authUID string) (userID string, err error)
}

type ctxKey struct{}

// UserID はコンテキストから内部ユーザーIDを取り出す。
func UserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ctxKey{}).(string)
	return id, ok && id != ""
}

// WithUserID はテスト等でユーザーIDをコンテキストに設定する。
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxKey{}, userID)
}

// NewInterceptor は全RPCで認証を必須にするインターセプターを返す。
func NewInterceptor(v Verifier, r UserResolver) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			token, ok := strings.CutPrefix(req.Header().Get("Authorization"), "Bearer ")
			if !ok || token == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing bearer token"))
			}
			authUID, err := v.Verify(ctx, token)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid token"))
			}
			userID, err := r.Resolve(ctx, authUID)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			return next(WithUserID(ctx, userID), req)
		}
	}
}
