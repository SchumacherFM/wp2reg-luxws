package luxws

import (
	"context"
	"errors"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// ErrNotRunning is the error returned when the websocket receiver goroutine is
// no longer running and no specific error is available.
var ErrNotRunning = errors.New("receiver not running")

// ErrBusy is the error returned when concurrent requests for sending a message
// are made.
var ErrBusy = errors.New("connection is busy")

// Option is the type of options for transports.
type Option func(*transport)

// WithLogFunc supplies a logging function to the transport. Received and sent
// messages are written as log messages.
func WithLogFunc(log *zap.Logger) Option {
	return func(t *transport) {
		t.log = log
	}
}

type websocketConn interface {
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	SetWriteDeadline(time.Time) error
	WriteMessage(int, []byte) error
	ReadMessage() (int, []byte, error)
	Close() error
}

// Transport is a wrapper for a LuxWS websocket connection.
type Transport struct {
	// The receiver goroutine keeps a reference to the transport, thus
	// preventing it from being garbage collected. To enable automatic
	// collection a finalizer must be set on another object, namely the
	// returned Transport.
	*transport
}

type transport struct {
	log *zap.Logger

	mu       sync.Mutex
	ws       websocketConn
	recvDone chan struct{}
	recvErr  error
	handler  *responseHandler
}

func newTransport(ws websocketConn, opts []Option) *Transport {
	t := &transport{
		ws:       ws,
		recvDone: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(t)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	wrapper := &Transport{t}

	// Launch asynchronous receiver to keep processing incoming messages (e.g.
	// ping).
	go t.receiver()

	return wrapper
}

// Dial connects to a LuxWS server. The address must have the format
// "<host>:<port>" (see net.JoinHostPort). Use the context to establish
// a timeout.
func Dial(ctx context.Context, address string, opts ...Option) (*Transport, error) {
	url := url.URL{
		Scheme: "ws",
		Host:   address,
	}

	dialer := *websocket.DefaultDialer
	dialer.HandshakeTimeout = 30 * time.Second
	dialer.Subprotocols = append(dialer.Subprotocols, "Lux_WS")

	ws, _, err := dialer.DialContext(ctx, url.String(), nil)
	if err != nil {
		return nil, err
	}

	return newTransport(ws, opts), nil
}

// LocalAddr returns the local network address.
func (t *transport) LocalAddr() net.Addr {
	return t.ws.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (t *transport) RemoteAddr() net.Addr {
	return t.ws.RemoteAddr()
}

// Close immediately closes the underlying network connection. Any blocked read
// or write operations will be unblocked and return errors.
func (t *transport) Close() error {
	t.mu.Lock()
	err := t.ws.Close()
	t.mu.Unlock()

	if err != nil {
		return err
	}

	// Wait for receiver to terminate
	<-t.recvDone

	t.mu.Lock()
	t.recvErr = net.ErrClosed
	t.mu.Unlock()

	return nil
}

func (t *transport) receiver() {
	defer close(t.recvDone)

	err := t.receiverLoop()

	if err == nil {
		err = ErrNotRunning
	}

	t.mu.Lock()
	t.recvErr = err
	t.mu.Unlock()
}

func (t *transport) receiverLoop() error {
	t.mu.Lock()
	ws := t.ws
	t.mu.Unlock()

	for {
		messageType, payload, err := ws.ReadMessage()
		if err != nil {
			return err
		}

		if t.log != nil && t.log.Level() == zap.DebugLevel {
			t.log.Debug(
				"Received message",
				zap.Int("type", messageType),
				zap.ByteString("payload", payload),
			)
		}
		if messageType == websocket.TextMessage && len(payload) > 0 {
			t.mu.Lock()
			handler := t.handler
			t.mu.Unlock()

			if handler != nil {
				handler.Handle(payload)
			}
		}
	}
}

func (t *transport) writeMessage(ctx context.Context, cmd string) error {
	const messageType = websocket.TextMessage

	if deadline, ok := ctx.Deadline(); ok {
		if err := t.ws.SetWriteDeadline(deadline); err != nil {
			return err
		}

		defer t.ws.SetWriteDeadline(time.Time{})
	}

	if t.log != nil && t.log.Level() == zap.DebugLevel {
		t.log.Debug(
			"Sending message",
			zap.Int("type", messageType),
			zap.String("command", cmd),
		)
	}

	if err := t.ws.WriteMessage(messageType, []byte(cmd)); err != nil {
		return err
	}

	return nil
}

func (t *transport) roundTrip(ctx context.Context, req string, handler *responseHandler) error {
	var err error

	t.mu.Lock()
	select {
	case <-t.recvDone:
		err = t.recvErr
	default:
		if t.handler == nil {
			t.handler = handler
		} else {
			err = ErrBusy
		}
	}
	t.mu.Unlock()

	if err != nil {
		return err
	}

	if err = t.writeMessage(ctx, req); err != nil {
		t.mu.Lock()
		t.handler = nil
		t.mu.Unlock()
		return err
	}

	select {
	case <-handler.Done():
		err = handler.Err()
		t.mu.Lock()
	case <-t.recvDone:
		t.mu.Lock()
		err = t.recvErr
	case <-ctx.Done():
		err = ctx.Err()
		t.mu.Lock()
	}
	t.handler = nil
	t.mu.Unlock()

	return err
}

// RoundTrip sends a request as a single message. All incoming messages are
// passed to the given handler function. If a response message is deemed an
// acceptable response the handler must return nil. If the message is not
// acceptable, but not an error, ErrIgnore can be returned by the handler. In
// all other cases an error must be returned.
func (t *transport) RoundTrip(ctx context.Context, req string, fn ResponseHandlerFunc) error {
	return t.roundTrip(ctx, req, newResponseHandler(fn))
}
