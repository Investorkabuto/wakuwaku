// Package system はサーバーの稼働確認を提供する。
package system

import (
	"context"

	"connectrpc.com/connect"
	wakuwakuv1 "github.com/investorkabuto/wakuwaku/server/internal/gen/wakuwaku/v1"
)

type Service struct {
	Version string
}

func (s *Service) Ping(context.Context, *connect.Request[wakuwakuv1.PingRequest]) (*connect.Response[wakuwakuv1.PingResponse], error) {
	return connect.NewResponse(&wakuwakuv1.PingResponse{Version: s.Version}), nil
}
