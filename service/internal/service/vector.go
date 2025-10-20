package service

import (
	"net/http"

	"klabel/internal/biz"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// VectorService exposes vector CRUD and search endpoints.
type VectorService struct {
	uc *biz.VectorUsecase
}

// NewVectorService constructs the transport service.
func NewVectorService(uc *biz.VectorUsecase) *VectorService {
	return &VectorService{uc: uc}
}

// RegisterHTTP wires handlers onto the given HTTP server.
func (s *VectorService) RegisterHTTP(server *kratoshttp.Server) {
	api := server.Route("/api")
	api.POST("/vectors", s.Ingest)
	api.PUT("/vectors/{id}", s.Update)
	api.DELETE("/vectors/{id}", s.Delete)
	api.GET("/vectors", s.List)
	api.POST("/vectors/search", s.QuerySimilar)
}

// Ingest handles vector upsert requests.
// IngestRequest matches the API payload for vector ingestion.
type IngestRequest struct {
	ID           string            `json:"id"`
	Symbol       string            `json:"symbol"`
	Close        float32           `json:"close"`
	MaxVectors   []float32         `json:"max_vectors"`
	MAVectors    []float32         `json:"ma_vectors"`
	PriceVectors []float32         `json:"price_vectors"`
	Metadata     map[string]string `json:"metadata"`
}

func (s *VectorService) Ingest(ctx kratoshttp.Context) error {
	var req IngestRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	point := toBizPoint(&req)
	id, err := s.uc.Ingest(ctx, point)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, IngestResponse{ID: id})
}

// Update updates an existing vector.
func (s *VectorService) Update(ctx kratoshttp.Context) error {
	var path DeleteRequest
	if err := ctx.BindVars(&path); err != nil {
		return err
	}
	var req UpdateRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	req.ID = path.ID
	point := toBizPoint(&IngestRequest{
		ID:           req.ID,
		Symbol:       req.Symbol,
		Close:        req.Close,
		MaxVectors:   req.MaxVectors,
		MAVectors:    req.MAVectors,
		PriceVectors: req.PriceVectors,
		Metadata:     req.Metadata,
	})
	if err := s.uc.Update(ctx, point); err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, UpdateResponse{ID: req.ID})
}

type IngestResponse struct {
	ID string `json:"id"`
}

type UpdateRequest struct {
	ID           string            `json:"id"`
	Symbol       string            `json:"symbol"`
	Close        float32           `json:"close"`
	MaxVectors   []float32         `json:"max_vectors"`
	MAVectors    []float32         `json:"ma_vectors"`
	PriceVectors []float32         `json:"price_vectors"`
	Metadata     map[string]string `json:"metadata"`
}

type UpdateResponse struct {
	ID string `json:"id"`
}

type DeleteRequest struct {
	ID string `json:"id" path:"id"`
}

type DeleteResponse struct {
	Success bool `json:"success"`
}

type ListRequest struct {
	Symbol string `json:"symbol" form:"symbol"`
	Limit  int    `json:"limit" form:"limit"`
	Offset int    `json:"offset" form:"offset"`
}

type ListResponse struct {
	Items []VectorPoint `json:"items"`
	Total int           `json:"total"`
}

type QuerySimilarRequest struct {
	Symbol       string         `json:"symbol"`
	TopK         int            `json:"top_k"`
	Close        float32        `json:"close"`
	MaxVectors   []float32      `json:"max_vectors"`
	MAVectors    []float32      `json:"ma_vectors"`
	PriceVectors []float32      `json:"price_vectors"`
	Metrics      []MetricWeight `json:"metrics"`
}

type MetricWeight struct {
	Name   string  `json:"name"`
	Weight float32 `json:"weight"`
}

type QuerySimilarResponse struct {
	Results []SearchResult `json:"results"`
}

type SearchResult struct {
	ID           string             `json:"id"`
	Symbol       string             `json:"symbol"`
	Score        float32            `json:"score"`
	MetricScores map[string]float32 `json:"metric_scores,omitempty"`
	Point        VectorPoint        `json:"point"`
}

type VectorPoint struct {
	ID           string            `json:"id"`
	Symbol       string            `json:"symbol"`
	Close        float32           `json:"close"`
	MaxVectors   []float32         `json:"max_vectors"`
	MAVectors    []float32         `json:"ma_vectors"`
	PriceVectors []float32         `json:"price_vectors"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

func toBizPoint(req *IngestRequest) *biz.VectorPoint {
	if req == nil {
		return &biz.VectorPoint{}
	}
	return &biz.VectorPoint{
		ID:           req.ID,
		Symbol:       req.Symbol,
		Close:        req.Close,
		MaxVectors:   req.MaxVectors,
		MAVectors:    req.MAVectors,
		PriceVectors: req.PriceVectors,
		Metadata:     req.Metadata,
	}
}

func fromBizPoint(point *biz.VectorPoint) VectorPoint {
	if point == nil {
		return VectorPoint{}
	}
	return VectorPoint{
		ID:           point.ID,
		Symbol:       point.Symbol,
		Close:        point.Close,
		MaxVectors:   point.MaxVectors,
		MAVectors:    point.MAVectors,
		PriceVectors: point.PriceVectors,
		Metadata:     point.Metadata,
	}
}

// Delete removes a vector point.
func (s *VectorService) Delete(ctx kratoshttp.Context) error {
	var req DeleteRequest
	if err := ctx.BindVars(&req); err != nil {
		return err
	}
	if err := s.uc.Delete(ctx, req.ID); err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, DeleteResponse{Success: true})
}

// List returns paginated vectors.
func (s *VectorService) List(ctx kratoshttp.Context) error {
	var req ListRequest
	if err := ctx.BindQuery(&req); err != nil {
		return err
	}
	points, total, err := s.uc.List(ctx, req.Symbol, req.Offset, req.Limit)
	if err != nil {
		return err
	}
	items := make([]VectorPoint, 0, len(points))
	for _, p := range points {
		items = append(items, fromBizPoint(p))
	}
	return ctx.JSON(http.StatusOK, ListResponse{
		Items: items,
		Total: total,
	})
}

// QuerySimilar triggers hybrid similarity search.
func (s *VectorService) QuerySimilar(ctx kratoshttp.Context) error {
	var req QuerySimilarRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	query := toBizPoint(&IngestRequest{
		Symbol:       req.Symbol,
		Close:        req.Close,
		MaxVectors:   req.MaxVectors,
		MAVectors:    req.MAVectors,
		PriceVectors: req.PriceVectors,
	})
	metrics := make([]biz.MetricWeight, 0, len(req.Metrics))
	for _, metric := range req.Metrics {
		metrics = append(metrics, biz.MetricWeight{Name: metric.Name, Weight: metric.Weight})
	}
	filter := biz.SearchFilter{Symbol: req.Symbol}
	results, err := s.uc.QuerySimilar(ctx, query, metrics, req.TopK, filter)
	if err != nil {
		return err
	}
	resp := QuerySimilarResponse{Results: make([]SearchResult, 0, len(results))}
	for _, res := range results {
		item := SearchResult{
			ID:     res.Point.ID,
			Symbol: res.Point.Symbol,
			Score:  res.Score,
			Point:  fromBizPoint(res.Point),
		}
		if len(res.MetricScores) > 0 {
			item.MetricScores = res.MetricScores
		}
		resp.Results = append(resp.Results, item)
	}
	return ctx.JSON(http.StatusOK, resp)
}
