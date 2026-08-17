package graph

import (
	"math"
	"math/rand"
)

const layoutExtent = 80.0

// assignEmbeddingLayout places nodes in 3D from article embeddings.
// WHY: semantic proximity should drive coordinates, not link forces.
// Articles with vectors get PCA coordinates; tags sit at the centroid of
// connected articles; remaining nodes inherit neighbor averages.
func assignEmbeddingLayout(graph *GraphData, pages []PageInfo) {
	if graph == nil || len(graph.Nodes) == 0 {
		return
	}

	vectorByID := make(map[string][]float32, len(pages))
	for _, page := range pages {
		if len(page.Vector) == 0 {
			continue
		}
		vectorByID[page.ID] = page.Vector
	}

	ids := make([]string, 0, len(vectorByID))
	vectors := make([][]float32, 0, len(vectorByID))
	for _, node := range graph.Nodes {
		vec, ok := vectorByID[node.ID]
		if !ok {
			continue
		}
		ids = append(ids, node.ID)
		vectors = append(vectors, vec)
	}

	positions := make(map[string][3]float64, len(graph.Nodes))
	if len(vectors) >= 2 {
		projected := projectPCA3D(vectors)
		for i, id := range ids {
			positions[id] = projected[i]
		}
	} else if len(vectors) == 1 {
		positions[ids[0]] = [3]float64{0, 0, 0}
	}

	neighbors := adjacency(graph)
	fillMissingPositions(graph, positions, neighbors)

	for i := range graph.Nodes {
		pos, ok := positions[graph.Nodes[i].ID]
		if !ok {
			pos = [3]float64{0, 0, 0}
		}
		graph.Nodes[i].X = roundCoord(pos[0])
		graph.Nodes[i].Y = roundCoord(pos[1])
		graph.Nodes[i].Z = roundCoord(pos[2])
	}
}

func adjacency(graph *GraphData) map[string][]string {
	out := make(map[string][]string, len(graph.Nodes))
	for _, edge := range graph.Edges {
		out[edge.Source] = append(out[edge.Source], edge.Target)
		out[edge.Target] = append(out[edge.Target], edge.Source)
	}
	return out
}

func fillMissingPositions(graph *GraphData, positions map[string][3]float64, neighbors map[string][]string) {
	// Prefer tag/page centroids from already-placed semantic nodes.
	for pass := 0; pass < 8; pass++ {
		progress := false
		for _, node := range graph.Nodes {
			if _, ok := positions[node.ID]; ok {
				continue
			}
			sum := [3]float64{}
			count := 0
			for _, nb := range neighbors[node.ID] {
				pos, ok := positions[nb]
				if !ok {
					continue
				}
				sum[0] += pos[0]
				sum[1] += pos[1]
				sum[2] += pos[2]
				count++
			}
			if count == 0 {
				continue
			}
			positions[node.ID] = [3]float64{sum[0] / float64(count), sum[1] / float64(count), sum[2] / float64(count)}
			progress = true
		}
		if !progress {
			break
		}
	}

	// Deterministic fallback so builds stay reproducible without embeddings.
	rng := rand.New(rand.NewSource(42))
	for _, node := range graph.Nodes {
		if _, ok := positions[node.ID]; ok {
			continue
		}
		theta := rng.Float64() * 2 * math.Pi
		phi := math.Acos(2*rng.Float64() - 1)
		r := layoutExtent * 0.35
		positions[node.ID] = [3]float64{
			r * math.Sin(phi) * math.Cos(theta),
			r * math.Sin(phi) * math.Sin(theta),
			r * math.Cos(phi),
		}
	}
}

// projectPCA3D reduces embedding vectors to 3D via matrix-free power-iteration PCA.
func projectPCA3D(vectors [][]float32) [][3]float64 {
	n := len(vectors)
	if n == 0 {
		return nil
	}
	dim := len(vectors[0])
	centered := make([][]float64, n)
	mean := make([]float64, dim)
	for _, vec := range vectors {
		for i, value := range vec {
			if i >= dim {
				break
			}
			mean[i] += float64(value)
		}
	}
	invN := 1 / float64(n)
	for i := range mean {
		mean[i] *= invN
	}
	for i, vec := range vectors {
		row := make([]float64, dim)
		limit := dim
		if len(vec) < limit {
			limit = len(vec)
		}
		for j := 0; j < limit; j++ {
			row[j] = float64(vec[j]) - mean[j]
		}
		centered[i] = row
	}

	components := make([][]float64, 0, 3)
	for c := 0; c < 3; c++ {
		comp := powerIterationComponent(centered, dim, components)
		if comp == nil {
			comp = make([]float64, dim)
			if c < dim {
				comp[c] = 1
			}
		}
		components = append(components, comp)
		deflate(centered, comp)
	}

	// Rebuild centered rows for final projection (deflation mutated the working copy).
	centered = make([][]float64, n)
	for i, vec := range vectors {
		row := make([]float64, dim)
		limit := dim
		if len(vec) < limit {
			limit = len(vec)
		}
		for j := 0; j < limit; j++ {
			row[j] = float64(vec[j]) - mean[j]
		}
		centered[i] = row
	}

	out := make([][3]float64, n)
	var maxAbs float64
	for i, row := range centered {
		for c, comp := range components {
			var score float64
			for j := 0; j < dim; j++ {
				score += row[j] * comp[j]
			}
			out[i][c] = score
			if abs := math.Abs(score); abs > maxAbs {
				maxAbs = abs
			}
		}
	}
	if maxAbs > 0 {
		scale := layoutExtent / maxAbs
		for i := range out {
			out[i][0] *= scale
			out[i][1] *= scale
			out[i][2] *= scale
		}
	}
	return out
}

func powerIterationComponent(rows [][]float64, dim int, previous [][]float64) []float64 {
	if len(rows) == 0 || dim == 0 {
		return nil
	}
	v := make([]float64, dim)
	rng := rand.New(rand.NewSource(int64(17 + 31*len(previous) + dim)))
	for i := range v {
		v[i] = rng.NormFloat64()
	}
	normalize(v)
	orthogonalize(v, previous)

	tmp := make([]float64, len(rows))
	next := make([]float64, dim)
	for iter := 0; iter < 40; iter++ {
		for i, row := range rows {
			var sum float64
			for j := 0; j < dim; j++ {
				sum += row[j] * v[j]
			}
			tmp[i] = sum
		}
		for j := 0; j < dim; j++ {
			var sum float64
			for i := range rows {
				sum += rows[i][j] * tmp[i]
			}
			next[j] = sum
		}
		orthogonalize(next, previous)
		if normalize(next) == 0 {
			return nil
		}
		copy(v, next)
	}
	return v
}

func deflate(rows [][]float64, component []float64) {
	for _, row := range rows {
		var score float64
		for j := range component {
			score += row[j] * component[j]
		}
		for j := range component {
			row[j] -= score * component[j]
		}
	}
}

func orthogonalize(v []float64, previous [][]float64) {
	for _, basis := range previous {
		var dot float64
		for i := range v {
			dot += v[i] * basis[i]
		}
		for i := range v {
			v[i] -= dot * basis[i]
		}
	}
}

func normalize(v []float64) float64 {
	var sum float64
	for _, value := range v {
		sum += value * value
	}
	if sum == 0 {
		return 0
	}
	inv := 1 / math.Sqrt(sum)
	for i := range v {
		v[i] *= inv
	}
	return 1
}

func roundCoord(v float64) float64 {
	return math.Round(v*1000) / 1000
}
