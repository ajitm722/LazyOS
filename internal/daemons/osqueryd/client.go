package osqueryd

import (
	"context"
	"fmt"
	"time"

	"github.com/ajitm722/LazyOS/internal/config"
	"github.com/ajitm722/LazyOS/internal/daemons"
	"github.com/ajitm722/LazyOS/internal/logger"
	goosquery "github.com/osquery/osquery-go"
)

type Client struct {
	client       *goosquery.ExtensionManagerClient
	queryTimeout time.Duration
	Schema       []daemons.TableSchema
}

func NewClient(socketPath string, startupTimeout time.Duration, queryTimeout time.Duration, schema []daemons.TableSchema) (*Client, error) {
	client, err := goosquery.NewClient(socketPath, startupTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to osquery socket: %w", err)
	}
	return &Client{client: client, queryTimeout: queryTimeout, Schema: schema}, nil
}

// NewClientFromConfig creates a Client using the osquery fields from the app
// config. Returns (nil, nil) when the socket path is empty, signalling the
// caller to skip this backend.
func NewClientFromConfig(cfg config.Config, schema []daemons.TableSchema) (*Client, error) {
	if cfg.OsquerySocket == "" {
		return nil, nil
	}
	return NewClient(cfg.OsquerySocket, cfg.OsqueryStartupTimeout, cfg.OsqueryQueryTimeout, schema)
}

func (c *Client) Close() error {
	if c.client != nil {
		c.client.Close()
	}
	return nil
}

// executeThriftQuery performs the raw Thrift RPC call and returns the result
// rows. Columns are not returned here; they are derived from the client's
// schema when rows == 0.
func (c *Client) executeThriftQuery(sql string) ([]map[string]string, error) {
	resp, err := c.client.Query(sql)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", daemons.ErrQueryFailed, err)
	}

	if resp.Status != nil && resp.Status.Code != 0 {
		return nil, fmt.Errorf("%w: status code %d, message: %s", daemons.ErrQueryFailed, resp.Status.Code, resp.Status.Message)
	}

	return resp.Response, nil
}

func (c *Client) Query(ctx context.Context, sql string) ([]map[string]string, []string, error) {
	log := logger.FromContext(ctx)
	log.Debug("osquery client query started", "sql", sql)

	queryCtx, cancel := context.WithTimeout(ctx, c.queryTimeout)
	defer cancel()

	type result struct {
		rows []map[string]string
		err  error
	}

	ch := make(chan result, 1)

	go func() {
		rows, err := c.executeThriftQuery(sql)
		ch <- result{rows: rows, err: err}
	}()

	select {
	case <-queryCtx.Done():
		log.Error("osquery client query timed out", "sql", sql)
		return nil, nil, fmt.Errorf("%w after %s for sql: %s", daemons.ErrQueryTimeout, c.queryTimeout, sql)

	case res := <-ch:
		if res.err != nil {
			log.Error("osquery client query failed", "sql", sql, "error", res.err)
			return nil, nil, res.err
		}

		var cols []string
		if len(res.rows) > 0 {
			for k := range res.rows[0] {
				cols = append(cols, k)
			}
		} else {
			cols = daemons.DeriveColumnsFromSchema(sql, c.Schema)
		}

		log.Debug("osquery client query completed", "sql", sql, "rows_returned", len(res.rows))
		return res.rows, cols, nil
	}
}
