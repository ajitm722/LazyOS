package kernel

import (
	"github.com/ajitm722/LazyOS/internal/config"
	"github.com/ajitm722/LazyOS/internal/daemons"
	"github.com/ajitm722/LazyOS/internal/daemons/osqueryd"
)

var _ daemons.Queryer = (*Queryer)(nil)

// Queryer wraps the osqueryd Thrift client and exposes kernel tables.
type Queryer struct {
	*osqueryd.Client
}

// InitFromConfig creates an osquery-backed Queryer with kernel tables.
func InitFromConfig(cfg config.Config) (string, daemons.Queryer, error) {
	c, err := osqueryd.NewClientFromConfig(cfg, KernelTables)
	if err != nil || c == nil {
		return "", nil, err
	}
	return "osquery-kernel", &Queryer{Client: c}, nil
}

// GetSchema returns the kernel table catalog.
func (q *Queryer) GetSchema() []daemons.TableSchema {
	return KernelTables
}
