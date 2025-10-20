package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"klabel/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

const (
	payloadKeySymbol       = "symbol"
	payloadKeyClose        = "close"
	payloadKeyMaxVectors   = "max_vectors"
	payloadKeyMAVectors    = "ma_vectors"
	payloadKeyPriceVectors = "price_vectors"
)

// vectorRepo persists vectors inside Qdrant over HTTP API.
type vectorRepo struct {
	data *Data
	log  *log.Helper
}

// NewVectorRepo wires data dependencies into biz layer.
func NewVectorRepo(data *Data, logger log.Logger) biz.VectorRepo {
	return &vectorRepo{data: data, log: log.NewHelper(logger)}
}

func (r *vectorRepo) UpsertPoint(ctx context.Context, point *biz.VectorPoint) (string, error) {
	if point == nil {
		return "", fmt.Errorf("point is nil")
	}
	if point.ID == "" {
		point.ID = uuid.NewString()
	}
	if err := r.ensureCollection(ctx, point); err != nil {
		return "", err
	}
	req := qdrantUpsertRequest{
		Points: []qdrantPoint{buildQdrantPoint(point)},
	}
	var resp qdrantOperationResponse
	if err := r.do(ctx, http.MethodPut, r.collectionPath("/points"), req, &resp); err != nil {
		return "", err
	}
	return point.ID, nil
}

func (r *vectorRepo) UpdatePoint(ctx context.Context, point *biz.VectorPoint) error {
	if point == nil || point.ID == "" {
		return fmt.Errorf("id is required")
	}
	_, err := r.UpsertPoint(ctx, point)
	return err
}

func (r *vectorRepo) DeletePoint(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("id is empty")
	}
	req := qdrantDeleteRequest{Points: []string{id}}
	var resp qdrantOperationResponse
	return r.do(ctx, http.MethodPost, r.collectionPath("/points/delete"), req, &resp)
}

func (r *vectorRepo) ListPoints(ctx context.Context, symbol string, offset, limit int) ([]*biz.VectorPoint, int, error) {
	if limit <= 0 {
		limit = 20
	}
	req := qdrantScrollRequest{
		Limit:       limit,
		WithPayload: true,
	}
	if symbol != "" {
		req.Filter = buildSymbolFilter(symbol)
	}
	var resp qdrantScrollResponse
	if err := r.do(ctx, http.MethodPost, r.collectionPath("/points/scroll"), req, &resp); err != nil {
		return nil, 0, err
	}
	points := make([]*biz.VectorPoint, 0, len(resp.Result))
	for _, item := range resp.Result {
		point := toBizPoint(item.Payload)
		point.ID = fmt.Sprintf("%v", item.ID)
		points = append(points, point)
	}
	total := len(points)
	return points, total, nil
}

func (r *vectorRepo) SearchByMetric(ctx context.Context, query *biz.VectorPoint, metric string, topK int, filter biz.SearchFilter) ([]*biz.SearchResult, error) {
	req := qdrantSearchRequest{
		Vector:      combineVectors(query),
		Limit:       topK,
		WithPayload: true,
		WithVector:  true,
	}
	if filter.Symbol != "" {
		req.Filter = buildSymbolFilter(filter.Symbol)
	}
	var resp qdrantSearchResponse
	if err := r.do(ctx, http.MethodPost, r.collectionPath("/points/search"), req, &resp); err != nil {
		return nil, err
	}
	results := make([]*biz.SearchResult, 0, len(resp.Result))
	dq := combineVectors(query)
	for _, item := range resp.Result {
		vector := toFloat32Slice(item.Vector)
		point := toBizPoint(item.Payload)
		point.ID = fmt.Sprintf("%v", item.ID)
		score := computeScore(metric, dq, vector, item.Score)
		results = append(results, &biz.SearchResult{
			Point: point,
			Score: score,
		})
	}
	return results, nil
}

func (r *vectorRepo) ensureCollection(ctx context.Context, point *biz.VectorPoint) error {
	dim := len(point.MaxVectors) + len(point.MAVectors) + len(point.PriceVectors)
	if dim == 0 {
		return fmt.Errorf("vector dimension is zero")
	}
	req := qdrantCreateCollectionRequest{
		Vectors: qdrantVectorParams{
			Size:     dim,
			Distance: "Cosine",
		},
	}
	var resp qdrantOperationResponse
	err := r.do(ctx, http.MethodPut, r.collectionPath(""), req, &resp)
	if err != nil {
		if isAlreadyExists(err) {
			return nil
		}
		return err
	}
	return nil
}

func (r *vectorRepo) do(ctx context.Context, method, path string, payload any, out any) error {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, r.data.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if r.data.apiKey != "" {
		req.Header.Set("api-key", r.data.apiKey)
	}
	resp, err := r.data.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		return fmt.Errorf("qdrant %s %s -> %d: %s", method, path, resp.StatusCode, buf.String())
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (r *vectorRepo) collectionPath(handler string) string {
	return fmt.Sprintf("/collections/%s%s", r.data.collection, handler)
}

func buildQdrantPoint(point *biz.VectorPoint) qdrantPoint {
	payload := map[string]any{
		payloadKeySymbol:       point.Symbol,
		payloadKeyClose:        point.Close,
		payloadKeyMaxVectors:   floatSliceToAny(point.MaxVectors),
		payloadKeyMAVectors:    floatSliceToAny(point.MAVectors),
		payloadKeyPriceVectors: floatSliceToAny(point.PriceVectors),
	}
	for k, v := range point.Metadata {
		payload[k] = v
	}
	return qdrantPoint{
		ID:      point.ID,
		Vector:  combineVectors(point),
		Payload: payload,
	}
}

func combineVectors(point *biz.VectorPoint) []float32 {
	totalLen := len(point.MaxVectors) + len(point.MAVectors) + len(point.PriceVectors)
	combined := make([]float32, 0, totalLen)
	combined = append(combined, point.MaxVectors...)
	combined = append(combined, point.MAVectors...)
	combined = append(combined, point.PriceVectors...)
	return combined
}

func floatSliceToAny(values []float32) []any {
	if len(values) == 0 {
		return nil
	}
	out := make([]any, len(values))
	for i, value := range values {
		out[i] = value
	}
	return out
}

func toBizPoint(payload map[string]any) *biz.VectorPoint {
	if payload == nil {
		return &biz.VectorPoint{}
	}
	return &biz.VectorPoint{
		Symbol:       stringValue(payload[payloadKeySymbol]),
		Close:        float32Value(payload[payloadKeyClose]),
		MaxVectors:   toFloat32Slice(payload[payloadKeyMaxVectors]),
		MAVectors:    toFloat32Slice(payload[payloadKeyMAVectors]),
		PriceVectors: toFloat32Slice(payload[payloadKeyPriceVectors]),
		Metadata:     mapStringAnyToString(payload),
	}
}

func stringValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func float32Value(v any) float32 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return float32(val)
	case float32:
		return val
	default:
		if s := stringValue(v); s != "" {
			if parsed, err := strconv.ParseFloat(s, 32); err == nil {
				return float32(parsed)
			}
		}
	}
	return 0
}

func toFloat32Slice(v any) []float32 {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []float32:
		return val
	case []float64:
		out := make([]float32, len(val))
		for i, item := range val {
			out[i] = float32(item)
		}
		return out
	case []any:
		out := make([]float32, 0, len(val))
		for _, item := range val {
			out = append(out, float32Value(item))
		}
		return out
	default:
		return nil
	}
}

func mapStringAnyToString(payload map[string]any) map[string]string {
	meta := make(map[string]string)
	for key, value := range payload {
		if key == payloadKeySymbol || key == payloadKeyClose || key == payloadKeyMaxVectors || key == payloadKeyMAVectors || key == payloadKeyPriceVectors {
			continue
		}
		meta[key] = stringValue(value)
	}
	if len(meta) == 0 {
		return nil
	}
	return meta
}

func computeScore(metric string, query, candidate []float32, qdrantScore float32) float32 {
	switch strings.ToLower(metric) {
	case "cosine":
		return qdrantScore
	case "manhattan":
		return -manhattanDistance(query, candidate)
	case "euclidean", "l2":
		return -euclideanDistance(query, candidate)
	default:
		return qdrantScore
	}
}

func manhattanDistance(a, b []float32) float32 {
	var sum float32
	length := min(len(a), len(b))
	for i := 0; i < length; i++ {
		diff := a[i] - b[i]
		if diff < 0 {
			diff = -diff
		}
		sum += diff
	}
	return sum
}

func euclideanDistance(a, b []float32) float32 {
	var sum float64
	length := min(len(a), len(b))
	for i := 0; i < length; i++ {
		diff := float64(a[i] - b[i])
		sum += diff * diff
	}
	return float32(math.Sqrt(sum))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func buildSymbolFilter(symbol string) *qdrantFilter {
	return &qdrantFilter{
		Must: []qdrantCondition{
			{
				Key: payloadKeySymbol,
				Match: qdrantMatch{
					Value: symbol,
				},
			},
		},
	}
}

func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "exists") || strings.Contains(err.Error(), "409")
}

// --- Qdrant DTOs ---

type qdrantPoint struct {
	ID      string         `json:"id"`
	Vector  []float32      `json:"vector"`
	Payload map[string]any `json:"payload,omitempty"`
}

type qdrantUpsertRequest struct {
	Points []qdrantPoint `json:"points"`
}

type qdrantDeleteRequest struct {
	Points []string `json:"points"`
}

type qdrantSearchRequest struct {
	Vector      []float32     `json:"vector"`
	Limit       int           `json:"limit"`
	Filter      *qdrantFilter `json:"filter,omitempty"`
	WithPayload bool          `json:"with_payload"`
	WithVector  bool          `json:"with_vector"`
}

type qdrantScrollRequest struct {
	Limit       int           `json:"limit"`
	Offset      string        `json:"offset,omitempty"`
	Filter      *qdrantFilter `json:"filter,omitempty"`
	WithPayload bool          `json:"with_payload"`
}

type qdrantCreateCollectionRequest struct {
	Vectors qdrantVectorParams `json:"vectors"`
}

type qdrantVectorParams struct {
	Size     int    `json:"size"`
	Distance string `json:"distance"`
}

type qdrantFilter struct {
	Must []qdrantCondition `json:"must"`
}

type qdrantCondition struct {
	Key   string      `json:"key"`
	Match qdrantMatch `json:"match"`
}

type qdrantMatch struct {
	Value string `json:"value"`
}

type qdrantOperationResponse struct {
	Status string  `json:"status"`
	Time   float64 `json:"time"`
}

type qdrantSearchResponse struct {
	Result []qdrantScoredPoint `json:"result"`
}

type qdrantScoredPoint struct {
	ID      any            `json:"id"`
	Score   float32        `json:"score"`
	Payload map[string]any `json:"payload"`
	Vector  any            `json:"vector"`
}

type qdrantScrollResponse struct {
	Result []qdrantPayloadPoint `json:"result"`
}

type qdrantPayloadPoint struct {
	ID      any            `json:"id"`
	Payload map[string]any `json:"payload"`
}
