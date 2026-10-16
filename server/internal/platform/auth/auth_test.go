package auth_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/investorkabuto/wakuwaku/server/internal/platform/auth"
)

type fakeResolver struct{}

func (fakeResolver) Resolve(_ context.Context, authUID string) (string, error) {
	return "user-" + authUID, nil
}

func TestDevVerifier(t *testing.T) {
	v := auth.DevVerifier{}
	if uid, err := v.Verify(context.Background(), "dev:hana"); err != nil || uid != "dev:hana" {
		t.Errorf("uid=%q err=%v", uid, err)
	}
	for _, tok := range []string{"", "dev:", "hana", "Bearer dev:hana"} {
		if _, err := v.Verify(context.Background(), tok); err == nil {
			t.Errorf("token %q accepted", tok)
		}
	}
}

func TestInterceptor(t *testing.T) {
	var gotUser string
	next := connect.UnaryFunc(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		gotUser, _ = auth.UserID(ctx)
		return nil, nil
	})
	call := auth.NewInterceptor(auth.DevVerifier{}, fakeResolver{})(next)

	tests := []struct {
		header   string
		wantCode connect.Code
		wantUser string
	}{
		{"", connect.CodeUnauthenticated, ""},
		{"Bearer ", connect.CodeUnauthenticated, ""},
		{"Bearer bogus", connect.CodeUnauthenticated, ""},
		{"Bearer dev:hana", 0, "user-dev:hana"},
	}
	for _, tt := range tests {
		gotUser = ""
		req := connect.NewRequest(&struct{}{})
		if tt.header != "" {
			req.Header().Set("Authorization", tt.header)
		}
		_, err := call(context.Background(), req)
		if tt.wantCode == 0 {
			if err != nil || gotUser != tt.wantUser {
				t.Errorf("%q: user=%q err=%v", tt.header, gotUser, err)
			}
		} else if connect.CodeOf(err) != tt.wantCode {
			t.Errorf("%q: err=%v, want %v", tt.header, err, tt.wantCode)
		}
	}
}
