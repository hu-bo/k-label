package biz

import (
	"context"
	"errors"
	"sort"

	"github.com/go-kratos/kratos/v2/log"
)

// VectorPoint represents a single persisted vector enriched with metadata.
type VectorPoint struct {
	ID           string
	Symbol       string
	Close        float32
	MaxVectors   []float32
	MAVectors    []float32
	PriceVectors []float32
	Metadata     map[string]string
}

// MetricWeight configures a hybrid search metric and its weight.
type MetricWeight struct {
	Name   string
	Weight float32
}

// SearchResult captures a hybrid similarity result.
type SearchResult struct {
	Point        *VectorPoint
	Score        float32
	MetricScores map[string]float32
}

// SearchFilter narrows down search space to a symbol when provided.
type SearchFilter struct {
	Symbol string
}

// VectorRepo defines persistence operations for vectors.
type VectorRepo interface {
	UpsertPoint(ctx context.Context, point *VectorPoint) (string, error)
	UpdatePoint(ctx context.Context, point *VectorPoint) error
	DeletePoint(ctx context.Context, id string) error
	ListPoints(ctx context.Context, symbol string, offset, limit int) ([]*VectorPoint, int, error)
	SearchByMetric(ctx context.Context, query *VectorPoint, metric string, topK int, filter SearchFilter) ([]*SearchResult, error)
}

// VectorUsecase orchestrates vector workflows.
type VectorUsecase struct {
	repo VectorRepo
	log  *log.Helper
}

// NewVectorUsecase constructs a new usecase instance.
func NewVectorUsecase(repo VectorRepo, logger log.Logger) *VectorUsecase {
	return &VectorUsecase{repo: repo, log: log.NewHelper(logger)}
}

// Ingest persists or updates a vector point with optional custom key.
func (uc *VectorUsecase) Ingest(ctx context.Context, point *VectorPoint) (string, error) {
	if point == nil {
		return "", errors.New("point is nil")
	}
	if point.Symbol == "" {
		return "", errors.New("symbol is required")
	}
	id, err := uc.repo.UpsertPoint(ctx, point)
	if err != nil {
		return "", err
	}
	uc.log.WithContext(ctx).Infof("ingested vector id=%s symbol=%s", id, point.Symbol)
	return id, nil
}

// Update modifies an existing vector.
func (uc *VectorUsecase) Update(ctx context.Context, point *VectorPoint) error {
	if point == nil || point.ID == "" {
		return errors.New("id is required")
	}
	if err := uc.repo.UpdatePoint(ctx, point); err != nil {
		return err
	}
	uc.log.WithContext(ctx).Infof("updated vector id=%s", point.ID)
	return nil
}

// Delete removes a vector by id.
func (uc *VectorUsecase) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("id is required")
	}
	if err := uc.repo.DeletePoint(ctx, id); err != nil {
		return err
	}
	uc.log.WithContext(ctx).Infof("deleted vector id=%s", id)
	return nil
}

// List enumerates vectors with pagination.
func (uc *VectorUsecase) List(ctx context.Context, symbol string, offset, limit int) ([]*VectorPoint, int, error) {
	if limit <= 0 {
		limit = 20
	}
	return uc.repo.ListPoints(ctx, symbol, offset, limit)
}

// QuerySimilar executes a hybrid similarity search and re-ranks the merged results.
func (uc *VectorUsecase) QuerySimilar(ctx context.Context, query *VectorPoint, metrics []MetricWeight, topK int, filter SearchFilter) ([]*SearchResult, error) {
	if query == nil {
		return nil, errors.New("query payload is required")
	}
	if len(metrics) == 0 {
		metrics = []MetricWeight{{Name: "cosine", Weight: 1}}
	}
	if topK <= 0 {
		topK = 10
	}

	metricTotals := make(map[string]float32, len(metrics))
	for _, m := range metrics {
		w := m.Weight
		if w <= 0 {
			w = 1
		}
		metricTotals[m.Name] = w
	}

	aggregate := make(map[string]*SearchResult)

	for _, metric := range metrics {
		results, err := uc.repo.SearchByMetric(ctx, query, metric.Name, topK, filter)
		if err != nil {
			return nil, err
		}
		weight := metric.Weight
		if weight <= 0 {
			weight = 1
		}
		for _, res := range results {
			if res == nil || res.Point == nil {
				continue
			}
			existing, ok := aggregate[res.Point.ID]
			if !ok {
				aggregate[res.Point.ID] = &SearchResult{
					Point:        res.Point,
					MetricScores: map[string]float32{metric.Name: res.Score},
					Score:        res.Score * weight,
				}
				continue
			}
			if existing.MetricScores == nil {
				existing.MetricScores = make(map[string]float32)
			}
			existing.MetricScores[metric.Name] = res.Score
			existing.Score += res.Score * weight
		}
	}

	merged := make([]*SearchResult, 0, len(aggregate))
	for _, item := range aggregate {
		merged = append(merged, item)
	}

	sort.SliceStable(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})

	if len(merged) > topK {
		merged = merged[:topK]
	}

	return merged, nil
}
