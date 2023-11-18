package luxwsclient

import (
	"context"
	"encoding/xml"

	"go.uber.org/zap"

	"github.com/hansmi/wp2reg-luxws/luxws"
)

func xmlUnmarshal(data []byte, v any) error {
	return xml.Unmarshal(data, v)
	//dec := xml.NewDecoder(bytes.NewReader(data))
	//// dec.CharsetReader = charset.NewReaderLabel
	//return dec.Decode(v)
}

// String stores s in a new string value and returns a pointer to it.
func String(s string) *string {
	return &s
}

type transport interface {
	RoundTrip(context.Context, string, luxws.ResponseHandlerFunc) error
	Close() error
}

// Option is the type of options for clients.
type Option func(*Client)

// WithLogFunc supplies a logging function to the client.
func WithLogFunc(log *zap.Logger) Option {
	return func(c *Client) {
		c.log = log
	}
}

// Client is a wrapper around an underlying LuxWS connection.
type Client struct {
	log *zap.Logger
	t   transport
}

// Dial connects to a LuxWS server. The address must have the format
// "<host>:<port>" (see net.JoinHostPort). Use the context to establish
// a timeout.
//
// IDs returned by the server are unique to each connection.
func Dial(ctx context.Context, address string, opts ...Option) (*Client, error) {
	var err error

	c := &Client{}

	for _, opt := range opts {
		opt(c)
	}

	if c.t, err = luxws.Dial(ctx, address, luxws.WithLogFunc(c.log)); err != nil {
		return nil, err
	}

	return c, nil
}

// Close closes the underlying network connection.
func (c *Client) Close() error {
	return c.t.Close()
}

// Login sends a "LOGIN" command. The navigation structure is returned.
func (c *Client) Login(ctx context.Context, password string) (result *NavRoot, err error) {
	return result, c.t.RoundTrip(ctx, "LOGIN;"+password, func(payload []byte) error {
		result, err = NewNavRoot(payload, "navigation")
		return err
	})
}

// Get sends a "GET" command. The page content is returned.
func (c *Client) Get(ctx context.Context, id string) (result *ContentRoot, err error) {
	return result, c.t.RoundTrip(ctx, "GET;"+id, func(payload []byte) error {
		result, err = NewContentRoot(payload, "content")
		return err
	})
}
