package data

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"helloworld/internal/conf"

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
	qcfg, err := parseQdrantConfig(q)
	if err != nil {
		return nil, nil, err
	}
	if qcfg.collection == "" {
		return nil, nil, errors.New("qdrant collection must not be empty")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	data := &Data{
		httpClient: client,
		baseURL:    strings.TrimRight(qcfg.baseURL, "/"),
		collection: qcfg.collection,
		apiKey:     qcfg.apiKey,
		log:        helper,
	}
	cleanup := func() {
		helper.Info("closing data resources")
	}
	return data, cleanup, nil
}

type qdrantConfig struct {
	baseURL    string
	collection string
	apiKey     string
}

func parseQdrantConfig(c *conf.Qdrant) (*qdrantConfig, error) {
	cfg := &qdrantConfig{
		baseURL:    "http://127.0.0.1:6333",
		collection: "vectors",
	}
	if c == nil {
		return cfg, nil
	}
	if c.BaseURL != "" {
		cfg.baseURL = strings.TrimRight(c.BaseURL, "/")
	}
	if c.Collection != "" {
		cfg.collection = c.Collection
	}
	if c.APIKey != "" {
		cfg.apiKey = c.APIKey
	}
	if _, err := url.Parse(cfg.baseURL); err != nil {
		return nil, err
	}
	return cfg, nil
}
