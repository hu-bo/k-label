package biz

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

type mockVectorRepo struct {
	points map[string]*VectorPoint
}

func newMockVectorRepo() *mockVectorRepo {
	return &mockVectorRepo{points: make(map[string]*VectorPoint)}
}

func (m *mockVectorRepo) UpsertPoint(ctx context.Context, point *VectorPoint) (string, error) {
	if point.ID == "" {
		point.ID = "mock-id"
	}
	m.points[point.ID] = point
	return point.ID, nil
}
func (m *mockVectorRepo) UpdatePoint(ctx context.Context, point *VectorPoint) error {
	if point.ID == "" {
		return nil
	}
	m.points[point.ID] = point
	return nil
}
func (m *mockVectorRepo) DeletePoint(ctx context.Context, id string) error {
	delete(m.points, id)
	return nil
}
func (m *mockVectorRepo) ListPoints(ctx context.Context, symbol string, offset, limit int) ([]*VectorPoint, int, error) {
	var res []*VectorPoint
	for _, p := range m.points {
		if symbol == "" || p.Symbol == symbol {
			res = append(res, p)
		}
	}
	total := len(res)
	if offset > total {
		return []*VectorPoint{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return res[offset:end], total, nil
}
func (m *mockVectorRepo) SearchByMetric(ctx context.Context, query *VectorPoint, metric string, topK int, filter SearchFilter) ([]*SearchResult, error) {
	var results []*SearchResult
	for _, p := range m.points {
		if filter.Symbol != "" && p.Symbol != filter.Symbol {
			continue
		}
		results = append(results, &SearchResult{
			Point:        p,
			Score:        1.0,
			MetricScores: map[string]float32{metric: 1.0},
		})
	}
	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

func TestVectorUsecase_BasicFlow(t *testing.T) {
	repo := newMockVectorRepo()
	uc := NewVectorUsecase(repo, log.NewStdLogger(nil))
	ctx := context.Background()
	// Ingest
	id, err := uc.Ingest(ctx, &VectorPoint{Symbol: "ETHUSDT", Close: 100})
	if err != nil || id == "" {
		t.Fatalf("Ingest failed: %v", err)
	}
	// Update
	err = uc.Update(ctx, &VectorPoint{ID: id, Symbol: "ETHUSDT", Close: 101})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	// List
	list, total, err := uc.List(ctx, "ETHUSDT", 0, 10)
	if err != nil || total == 0 {
		t.Fatalf("List failed: %v", err)
	}
	// QuerySimilar
	results, err := uc.QuerySimilar(ctx, &VectorPoint{Symbol: "ETHUSDT"}, []MetricWeight{{Name: "cosine", Weight: 1}}, 5, SearchFilter{Symbol: "ETHUSDT"})
	if err != nil || len(results) == 0 {
		t.Fatalf("QuerySimilar failed: %v", err)
	}
	// Delete
	err = uc.Delete(ctx, id)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}
