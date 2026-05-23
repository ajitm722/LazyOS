package mock

import (
	"context"
	"fmt"
	"time"

	"github.com/ajitm722/lazyos/internal/daemons"
)

var _ daemons.Queryer = (*MockQueryer)(nil)

type MockQueryer struct {
	Results           map[string][]map[string]string
	DefaultResult     []map[string]string
	DefaultErr        error
	SimulateSlowQuery bool
	SlowDuration      time.Duration
	InternalTimeout   time.Duration
	Schema            []daemons.TableSchema
}

func (m *MockQueryer) Query(ctx context.Context, sql string) ([]map[string]string, []string, error) {
	if m.SimulateSlowQuery && m.SlowDuration > 0 {
		queryCtx := ctx
		if m.InternalTimeout > 0 {
			var cancel context.CancelFunc
			queryCtx, cancel = context.WithTimeout(ctx, m.InternalTimeout)
			defer cancel()
		}
		select {
		case <-queryCtx.Done():
			return nil, nil, fmt.Errorf("%w: %v", daemons.ErrQueryTimeout, queryCtx.Err())
		case <-time.After(m.SlowDuration):
			return nil, nil, fmt.Errorf("mock exceeded simulated time without application context cancellation")
		}
	}
	if m.DefaultErr != nil {
		return nil, nil, m.DefaultErr
	}

	var rows []map[string]string
	if m.Results != nil {
		if r, ok := m.Results[sql]; ok {
			rows = r
		}
	}
	if rows == nil && m.DefaultResult != nil {
		rows = m.DefaultResult
	}
	if rows == nil {
		rows = []map[string]string{}
	}

	var cols []string
	if len(rows) > 0 {
		// Use columns as returned by the mock data
		for k := range rows[0] {
			cols = append(cols, k)
		}
	} else {
		// 0 rows: fall back to schema for column metadata
		cols = daemons.DeriveColumnsFromSchema(sql, m.Schema)
	}

	return rows, cols, nil
}

func (m *MockQueryer) Close() error {
	return nil
}

func (m *MockQueryer) GetSchema() []daemons.TableSchema {
	if m.Schema != nil {
		return m.Schema
	}
	return []daemons.TableSchema{}
}
