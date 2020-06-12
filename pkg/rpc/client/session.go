package client

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"time"

	"google.golang.org/grpc"
)

// NodeClient is a wrapper for the grpc node service client. It takes care of
// handling the session on behalf of the caller by maintaining a grpc
// interceptor
type NodeClient struct {
	pk ed25519.PublicKey
	sk ed25519.PrivateKey

	addr           string
	proto          string
	persistentConn *grpc.ClientConn

	ctx     context.Context
	cancel  context.CancelFunc
	errChan chan error

	authClient     *AuthClient
	sessionHandler *AuthClientInterceptor
}

// New creates a new NodeClient which takes care of maintaining the session
func New(proto, addr string) *NodeClient {

	// create the client
	pk, sk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	nc := &NodeClient{
		sessionHandler: NewClientInterceptor(pk, sk),
		addr:           addr,
		proto:          proto,
		pk:             pk,
		sk:             sk,
		ctx:            ctx,
		cancel:         cancel,
		errChan:        make(chan error, 1),
	}

	go nc.monitorError()
	return nc
}

// GetSessionConn triggers a CreateSession through the auth client and
// implicitly populates the interceptor with the session token.
// It returns a new connection or an error
func (n *NodeClient) GetSessionConn() (*grpc.ClientConn, error) {
	if n.IsSessionActive() {
		return n.persistentConn, nil
	}

	// session is not active but we still have a connection dangling around
	if n.persistentConn != nil {
		_ = n.persistentConn.Close()
	}

	// recreating the session and the connection
	conn, err := n.createConn()
	if err != nil {
		return nil, err
	}

	n.authClient = NewClient(conn, n.pk, n.sk)
	_, err = n.authClient.CreateSession()

	n.persistentConn = conn
	return conn, err
}

// ScheduleSessionRefresh refreshes the session with the specified cadence
func (n *NodeClient) ScheduleSessionRefresh(cadence time.Time) error {
	if !n.IsSessionActive() {
		return errors.New("no active session found. Please create a session first")
	}

	go ScheduleRefreshToken(n.ctx, n.authClient, n.sessionHandler, cadence, 3, n.errChan)
	return nil
}

// DropSession closes a session, invalidate the session token and closes the
// eventual persistent connection if any
func (n *NodeClient) DropSession() error {
	var conn *grpc.ClientConn
	var err error
	defer func() {
		// closing the existing connection
		_ = conn.Close()
		// if persistentConn is conn, it gets closed with the above instruction
		// otherwise all it happends is that it gets harmlessly put to nil
		n.persistentConn = nil
	}()

	if n.persistentConn != nil {
		conn = n.persistentConn
	} else {
		conn, err = n.createConn()
		if err != nil {
			return err
		}
	}

	authClient := NewClient(conn, n.pk, n.sk)
	err = authClient.DropSession()
	n.sessionHandler.invalidateToken()
	return err
}

func (n *NodeClient) monitorError() {
	err := <-n.errChan
	log.WithError(err).Warnln("problem in session handling, closing the client")
	n.Close()
}

// GracefulClose cancels the session refresh and actively drops the session
func (n *NodeClient) GracefulClose() {
	n.Close()
	if n.IsSessionActive() {
		_ = n.DropSession()
	}
}

// Close the client without notifying the server to close the session
func (n *NodeClient) Close() {
	n.cancel()
}

// IsSessionActive returns whether the session is active or otherwise
func (n *NodeClient) IsSessionActive() bool {
	return n.sessionHandler.isSessionActive()
}

func (n *NodeClient) createConn() (*grpc.ClientConn, error) {
	// create the GRPC connection
	return grpc.Dial(
		n.addr,
		grpc.WithInsecure(),
		grpc.WithContextDialer(getDialer(n.proto)),
		grpc.WithUnaryInterceptor(n.sessionHandler.Unary()),
	)
}

func getDialer(proto string) func(context.Context, string) (net.Conn, error) {
	d := &net.Dialer{}
	return func(ctx context.Context, addr string) (net.Conn, error) {
		return d.DialContext(ctx, proto, addr)
	}
}
