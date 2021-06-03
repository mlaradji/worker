package service

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	log "github.com/sirupsen/logrus"
)

const userIdKey = ContextKey("userId")

var (
	clientUserMap = map[ClientId]string{
		{Issuer: "CN=CA,OU=CA,O=Mohamed\\, Inc.,L=Vancouver,ST=British Columbia,C=CA", Subject: "CN=Client 1,OU=Client,O=Mohamed\\, Inc.,L=Vancouver,ST=British Columbia,C=CA"}: "client1",
		{Issuer: "CN=CA,OU=CA,O=Mohamed\\, Inc.,L=Vancouver,ST=British Columbia,C=CA", Subject: "CN=Client 2,OU=Client,O=Mohamed\\, Inc.,L=Vancouver,ST=British Columbia,C=CA"}: "client2",
	} // clientUserMap maps certificate Issuer+Subject to userId. Clients not in this map will not be allowed access.
)

// ClientID uniquely identifies a client, and is composed of the client certificate's issue and subject.
type ClientId struct {
	Issuer  string
	Subject string
}

// ContextKey simply wraps around a string value, allowing us to avoid setting built-in type context keys.
type ContextKey string

// clientIdFromContext extracts the client ID from a request context.
func clientIdFromContext(ctx context.Context) (ClientId, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ClientId{}, errors.New("unable to determine peer")
	}

	tlsAuth, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return ClientId{}, errors.New("invalid TLS credentials")
	}

	cert := tlsAuth.State.VerifiedChains[0][0]
	clientId := ClientId{Issuer: cert.Issuer.ToRDNSequence().String(), Subject: cert.Subject.ToRDNSequence().String()}

	return clientId, nil
}

// setUserIdInContext creates a new context with the passed user ID attached.
func setUserIdInContext(ctx context.Context, userId string) context.Context {
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

// UnaryAuth is a unary gRPC interceptor that authenticates and authorizes clients, and attaches their user ID to the request context.
func UnaryAuth(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	userId, err := authorize(ctx)
	if err != nil {
		return nil, err
	}

	// attach userId to context
	ctxUser := setUserIdInContext(ctx, userId)

	return handler(ctxUser, req)
}

// StreamAuth is a server stream gRPC interceptor that authenticates and authorizes clients, and attaches their user ID to the request context.
func StreamAuth(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	userId, err := authorize(ss.Context())
	if err != nil {
		return err
	}

	// attach userId to context
	ctxUser := setUserIdInContext(ss.Context(), userId)
	sswc := NewServerStreamWithContext(ctxUser, ss)

	return handler(srv, sswc)
}

// authorize gets the client ID from the context, checks it against the global user map, and returns the userId.
func authorize(ctx context.Context) (string, error) {
	logger := log.WithFields(log.Fields{"func": "authenticateAndAuthorize"})

	// client authentication
	clientId, err := clientIdFromContext(ctx)
	if err != nil {
		logger.WithError(err).Error("unable to get client id from context")
		return "", status.Error(codes.Unauthenticated, "unable to determine client id")
	}

	// client authorization
	userId, ok := clientUserMap[clientId]
	if !ok {
		logger.WithField("clientId", clientId).Debug("client is not authorized to access resource")
		return "", status.Error(codes.PermissionDenied, "client is not authorized to access resource")
	}

	return userId, nil
}
