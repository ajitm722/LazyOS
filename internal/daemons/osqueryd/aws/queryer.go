package aws

import (
	"github.com/ajitm722/LazyOS/internal/config"
	"github.com/ajitm722/LazyOS/internal/daemons"
	"github.com/ajitm722/LazyOS/internal/daemons/osqueryd"
)

var _ daemons.Queryer = (*Queryer)(nil)

// Queryer wraps the osqueryd Thrift client and exposes AWS cloudquery tables.
type Queryer struct {
	*osqueryd.Client
}

// InitFromConfig creates an osquery-backed Queryer with AWS tables.
func InitFromConfig(cfg config.Config) (string, daemons.Queryer, error) {
	c, err := osqueryd.NewClientFromConfig(cfg, AWSTables)
	if err != nil || c == nil {
		return "", nil, err
	}
	return "osquery-aws", &Queryer{Client: c}, nil
}

// GetSchema returns the AWS table catalog.
func (q *Queryer) GetSchema() []daemons.TableSchema {
	return AWSTables
}
