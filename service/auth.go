package service

import (
	"context"
	"errors"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

const userIdKey = ContextKey("userId")

// ClientID uniquely identifies a client, and is composed of the client certificate's issue and subject.
type ClientId struct {
	Issuer  string
	Subject string
}

// ContextKey simply wraps around a string value, allowing us to avoid setting built-in type context keys.
type ContextKey string

// ClientIdFromContext extracts the client ID from a request context.
func ClientIdFromContext(ctx context.Context) (ClientId, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ClientId{}, errors.New("unable to determine peer")
	}

	tlsAuth, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return ClientId{}, errors.New("invalid TLS credentials")
	}

	cert := tlsAuth.State.VerifiedChains[0][0]
	clientId := ClientId{Issuer: cert.Issuer.String(), Subject: cert.Subject.String()}
	return clientId, nil
}

// SetUserIdInContext creates a new context with the passed user ID attached.
func SetUserIdInContext(ctx context.Context, userId string) context.Context {
	return context.WithValue(ctx, userIdKey, userId)
}

//GetUserIdFromContext retrieves the user id value from the context, returning an error if it is missing or invalid.
func GetUserIdFromContext(ctx context.Context) (string, error) {
	userId, ok := ctx.Value(userIdKey).(string)
	if !ok {
		return "", errors.New("client id is missing or invalid")
	}

	return userId, nil
}
