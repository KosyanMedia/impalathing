package impalathing

import (
	"context"
	"fmt"
	"time"

	"github.com/KosyanMedia/impalathing/services/beeswax"
	impala "github.com/KosyanMedia/impalathing/services/impalaservice"
	"github.com/apache/thrift/lib/go/thrift"
)

type Options struct {
	PollIntervalSeconds float64
	BatchSize           int64
}

var (
	DefaultOptions = Options{PollIntervalSeconds: 0.1, BatchSize: 10000}
)

type Connection struct {
	client    *impala.ImpalaServiceClient
	handle    *beeswax.QueryHandle
	transport thrift.TTransport
	options   Options
}

func Connect(host string, port int, options Options, saslConfiguration map[string]string) (*Connection, error) {
	socket, err := thrift.NewTSocketTimeout(fmt.Sprintf("%s:%d", host, port), 10000*time.Millisecond)

	if err != nil {
		return nil, err
	}

	var transport thrift.TTransport
	if _, ok := saslConfiguration["mechanismName"]; ok {
		saslConfiguration["service"] = "impala"
		transport, err = NewTSaslTransport(socket, host, saslConfiguration["mechanismName"], saslConfiguration)
		if err != nil {
			return nil, err
		}
	} else {
		transportFactory := thrift.NewTBufferedTransportFactory(24 * 1024 * 1024)
		transport, _ = transportFactory.GetTransport(socket)
	}
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()

	if err := transport.Open(); err != nil {
		return nil, err
	}

	client := impala.NewImpalaServiceClientFactory(transport, protocolFactory)

	return &Connection{client, nil, transport, options}, nil
}

func (c *Connection) isOpen() bool {
	return c.client != nil
}

func (c *Connection) Close() error {
	if c.isOpen() {
		if c.handle != nil {
			_, err := c.client.Cancel(context.Background(), c.handle)
			if err != nil {
				return err
			}
			c.handle = nil
		}

		c.transport.Close()
		c.client = nil
	}
	return nil
}

func (c *Connection) query(ctx context.Context, query string) (RowSet, error) {
	bquery := beeswax.Query{}

	bquery.Query = query
	bquery.Configuration = []string{}

	handle, err := c.client.Query(ctx, &bquery)

	if err != nil {
		return nil, err
	}

	return newRowSet(c.client, handle, c.options), nil
}

func (c *Connection) Query(query string) (RowSet, error) {
	return c.query(context.Background(), query)
}

func (c *Connection) QueryWithContext(ctx context.Context, query string) (RowSet, error) {
	return c.query(ctx, query)
}
