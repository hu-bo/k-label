package data

import (
	"net/http"
	"strings"
	"time"

	"klabel/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewVectorRepo)

// Data wraps outbound dependencies such as Qdrant.
type Data struct {
	httpClient *http.Client
	baseURL    string
	collection string
	apiKey     string
	log        *log.Helper
}

// NewData builds the data layer dependencies.
func NewData(c *conf.Data, q *conf.Qdrant, logger log.Logger) (*Data, func(), error) {
	helper := log.NewHelper(logger)
	client := &http.Client{Timeout: 10 * time.Second}
	data := &Data{
		httpClient: client,
		baseURL:    strings.TrimRight(c.Qdrant.BaseUrl, "/"),
		collection: c.Qdrant.Collection,
		apiKey:     c.Qdrant.ApiKey,
		log:        helper,
	}
	cleanup := func() {
		helper.Info("closing data resources")
	}
	return data, cleanup, nil
}
