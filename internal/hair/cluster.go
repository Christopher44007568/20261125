package hair

import (
	"fmt"
	"math"
	"math/rand"
)

const (
	featureSamples = 4
	featureDim     = featureSamples * 3
	maxIterations  = 7
)

func BuildGuideLevels(asset *Asset, guideCounts []int, seed int64) {
	previous := asset.Levels[0].Entries
	for i, requested := range guideCounts {
		target := requested
		if target > len(previous) {
			target = len(previous)
		}
		if target < 1 {
			continue
		}
		entries := ClusterEntries(previous, target, seed+int64((i+1)*7919))
		asset.Levels = append(asset.Levels, Level{
			Index:     i + 1,
			Label:     fmt.Sprintf("guides-%d", len(entries)),
			Entries:   entries,
			TotalMass: sumWeights(entries),
		})
		previous = entries
	}
	if len(asset.Levels) > 1 {
		asset.GuideCount = len(asset.Levels[1].Entries)
	}
}

func ClusterEntries(entries []Entry, target int, seed int64) []Entry {
	if target >= len(entries) {
		out := make([]Entry, len(entries))
		copy(out, entries)
		for i := range out {
			out[i].ID = i
		}
		return out
	}

	features := make([][]float64, len(entries))
	for i := range entries {
		features[i] = strandFeature(entries[i].Points)
	}
	rng := rand.New(rand.NewSource(seed))
	centers := initializeCenters(features, entries, target, rng)
	assign := make([]int, len(entries))
	for i := range assign {
		assign[i] = -1
	}

	for iteration := 0; iteration < maxIterations; iteration++ {
		changed := assignNearest(features, centers, assign)
		masses := make([]float64, target)
		sums := make([][]float64, target)
		for i := range sums {
			sums[i] = make([]float64, featureDim)
		}
		for i, f := range features {
			c := assign[i]
			w := entries[i].Weight
			masses[c] += w
			for d := 0; d < featureDim; d++ {
				sums[c][d] += f[d] * w
			}
		}
		for c := 0; c < target; c++ {
			if masses[c] > 1e-12 {
				for d := 0; d < featureDim; d++ {
					centers[c][d] = sums[c][d] / masses[c]
				}
				continue
			}
			respawnEmptyCenter(features, centers, assign, c)
		}
		if iteration > 0 && changed < len(entries)/200+1 {
			break
		}
	}
	assignNearest(features, centers, assign)
	ensureNonEmpty(features, centers, assign)

	return buildClusterEntries(entries, features, centers, assign, target)
}

func strandFeature(points []Vec3) []float64 {
	f := make([]float64, featureDim)
	last := len(points) - 1
	for s := 0; s < featureSamples; s++ {
		idx := int(math.Round(float64(s) * float64(last) / float64(featureSamples-1)))
		p := points[idx]
		f[s*3] = p.X
		f[s*3+1] = p.Y
		f[s*3+2] = p.Z
	}
	return f
}

func initializeCenters(features [][]float64, entries []Entry, target int, rng *rand.Rand) [][]float64 {
	centers := make([][]float64, 0, target)
	first := weightedIndex(entries, rng)
	centers = append(centers, append([]float64(nil), features[first]...))
	nearest := make([]float64, len(features))
	for i := range nearest {
		nearest[i] = math.Inf(1)
	}

	for len(centers) < target {
		last := centers[len(centers)-1]
		var total float64
		for i, f := range features {
			d := squaredDistance(f, last)
			if d < nearest[i] {
				nearest[i] = d
			}
			total += entries[i].Weight * nearest[i]
		}

		var chosen int
		if total <= 1e-15 {
			chosen = rng.Intn(len(features))
		} else {
			r := rng.Float64() * total
			chosen = len(features) - 1
			var accum float64
			for i := range features {
				accum += entries[i].Weight * nearest[i]
				if accum >= r {
					chosen = i
					break
				}
			}
		}
		centers = append(centers, append([]float64(nil), features[chosen]...))
	}
	return centers
}

func weightedIndex(entries []Entry, rng *rand.Rand) int {
	var total float64
	for _, e := range entries {
		total += e.Weight
	}
	r := rng.Float64() * total
	var accum float64
	for i, e := range entries {
		accum += e.Weight
		if accum >= r {
			return i
		}
	}
	return len(entries) - 1
}

func assignNearest(features, centers [][]float64, assign []int) int {
	changed := 0
	for i, f := range features {
		best := 0
		bestDist := squaredDistance(f, centers[0])
		for c := 1; c < len(centers); c++ {
			d := squaredDistance(f, centers[c])
			if d < bestDist {
				bestDist = d
				best = c
			}
		}
		if assign[i] != best {
			assign[i] = best
			changed++
		}
	}
	return changed
}

func squaredDistance(a, b []float64) float64 {
	var sum float64
	for i := 0; i < featureDim; i++ {
		d := a[i] - b[i]
		sum += d * d
	}
	return sum
}

func respawnEmptyCenter(features, centers [][]float64, assign []int, empty int) {
	worst := -1
	var worstDist float64
	for i, f := range features {
		c := assign[i]
		if c < 0 || c == empty {
			continue
		}
		d := squaredDistance(f, centers[c])
		if d > worstDist {
			worstDist = d
			worst = i
		}
	}
	if worst < 0 {
		worst = empty % len(features)
	}
	copy(centers[empty], features[worst])
	assign[worst] = empty
}

func ensureNonEmpty(features, centers [][]float64, assign []int) {
	target := len(centers)
	counts := make([]int, target)
	for _, c := range assign {
		if c >= 0 {
			counts[c]++
		}
	}
	for empty := 0; empty < target; empty++ {
		if counts[empty] > 0 {
			continue
		}
		donor := -1
		moved := -1
		bestDist := -1.0
		for c := 0; c < target; c++ {
			if counts[c] < 2 {
				continue
			}
			for i, assigned := range assign {
				if assigned != c {
					continue
				}
				d := squaredDistance(features[i], centers[c])
				if d > bestDist {
					bestDist = d
					donor = c
					moved = i
				}
			}
		}
		if moved < 0 {
			continue
		}
		counts[donor]--
		counts[empty]++
		assign[moved] = empty
		copy(centers[empty], features[moved])
	}
}

func buildClusterEntries(entries []Entry, features, centers [][]float64, assign []int, target int) []Entry {
	members := make([][]int, target)
	out := make([]Entry, target)
	for i, c := range assign {
		members[c] = append(members[c], i)
		out[c].Weight += entries[i].Weight
	}

	for c := range out {
		if len(members[c]) == 0 {
			continue
		}
		rep := members[c][0]
		repDist := squaredDistance(features[rep], centers[c])
		for _, idx := range members[c][1:] {
			if d := squaredDistance(features[idx], centers[c]); d < repDist {
				rep = idx
				repDist = d
			}
		}
		sourceIDs := make([]int, 0, len(members[c]))
		for _, idx := range members[c] {
			sourceIDs = append(sourceIDs, entries[idx].ID)
		}
		out[c].ID = c
		out[c].Points = entries[rep].Points
		out[c].RepresentativeID = entries[rep].RepresentativeID
		out[c].RepresentativeLevel = entries[rep].RepresentativeLevel
		out[c].SourceIDs = sourceIDs
	}
	return out
}

func sumWeights(entries []Entry) float64 {
	var sum float64
	for _, e := range entries {
		sum += e.Weight
	}
	return sum
}
