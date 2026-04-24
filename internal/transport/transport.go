package transport

import "context"

type CCPolicy string

const (
	CCPolicyDefault CCPolicy = "default"
	CCPolicyBrutal  CCPolicy = "brutal"
)

type Session interface {
	SendReliable(ctx context.Context, payload []byte) error
	ReceiveReliable(ctx context.Context) ([]byte, error)
	SendUnreliable(ctx context.Context, payload []byte) error
	ReceiveUnreliable(ctx context.Context) ([]byte, error)
	Close() error
}

type Provider interface {
	Open(ctx context.Context, clientID string) (Session, error)
}

type BrutalConfig struct {
	TargetBPS int64
	RTTMs     int
}
