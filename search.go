package main

import "fmt"

// Trace runs the configured parallax search. Runtime failures always return
// the original geometry UV; no frame cache or previous coordinate is used.
func Trace(cfg Config, field HeightField, ray Ray) Result {
	cfg.normalize()
	if err := cfg.Validate(); err != nil {
		return fallbackResult(cfg, ray, 0, err.Error())
	}
	if field == nil {
		return fallbackResult(cfg, ray, 0, "nil height field")
	}
	if ray.MaxDepth == 0 {
		ray.MaxDepth = 1
	}
	if ray.MaxDepth <= 0 {
		return fallbackResult(cfg, ray, 0, fmt.Sprintf("max depth must be positive, got %g", ray.MaxDepth))
	}

	iterations := 0
	previous, err := sampleRay(cfg, field, ray, 0, &iterations)
	if err != nil {
		return fallbackResult(cfg, ray, iterations, err.Error())
	}
	if previous.gap >= -cfg.HitEpsilon {
		return previous.result(cfg, 0, iterations, 0, 0)
	}

	for layer := 1; layer <= cfg.Layers; layer++ {
		if iterations >= cfg.MaxIterations {
			return fallbackResult(cfg, ray, iterations,
				"maximum iteration budget reached before completing fixed steps")
		}

		depth := ray.MaxDepth * float64(layer) / float64(cfg.Layers)
		current, sampleErr := sampleRay(cfg, field, ray, depth, &iterations)
		if sampleErr != nil {
			return fallbackResult(cfg, ray, iterations, sampleErr.Error())
		}

		if current.gap >= -cfg.HitEpsilon {
			return resolveCrossing(cfg, field, ray, previous, current, layer, iterations)
		}
		previous = current
	}

	return fallbackResult(cfg, ray, iterations,
		fmt.Sprintf("no intersection through depth %g", ray.MaxDepth))
}

func resolveCrossing(cfg Config, field HeightField, ray Ray, outside, inside sample, layer, iterations int) Result {
	switch cfg.Mode {
	case ModeFixed:
		return inside.result(cfg, layer, iterations, 0, inside.depth-outside.depth)
	case ModeBinary:
		return fixedBinaryRefinement(cfg, field, ray, outside, inside, layer, iterations)
	case ModeMaxIteration:
		return refineToLimit(cfg, field, ray, outside, inside, layer, iterations)
	default:
		return fallbackResult(cfg, ray, iterations, fmt.Sprintf("unknown search mode %q", cfg.Mode))
	}
}

func fixedBinaryRefinement(cfg Config, field HeightField, ray Ray, outside, inside sample, layer, iterations int) Result {
	lo, hi := outside, inside
	used := 0
	for used < cfg.BinaryIterations {
		if iterations >= cfg.MaxIterations {
			return fallbackResult(cfg, ray, iterations,
				"maximum iteration budget reached during binary refinement")
		}

		midDepth := (lo.depth + hi.depth) / 2
		mid, err := sampleRay(cfg, field, ray, midDepth, &iterations)
		if err != nil {
			return fallbackResult(cfg, ray, iterations, err.Error())
		}
		used++

		if mid.gap >= -cfg.HitEpsilon {
			hi = mid
		} else {
			lo = mid
		}
	}

	result := hi.result(cfg, layer, iterations, used, hi.depth-lo.depth)
	return result
}

func refineToLimit(cfg Config, field HeightField, ray Ray, outside, inside sample, layer, iterations int) Result {
	lo, hi := outside, inside
	used := 0
	for hi.depth-lo.depth > cfg.Tolerance {
		if iterations >= cfg.MaxIterations {
			return fallbackResult(cfg, ray, iterations, fmt.Sprintf(
				"maximum iterations reached with bracket width %.3g > tolerance %.3g",
				hi.depth-lo.depth, cfg.Tolerance))
		}

		midDepth := (lo.depth + hi.depth) / 2
		mid, err := sampleRay(cfg, field, ray, midDepth, &iterations)
		if err != nil {
			return fallbackResult(cfg, ray, iterations, err.Error())
		}
		used++

		if mid.gap >= -cfg.HitEpsilon {
			hi = mid
		} else {
			lo = mid
		}
	}

	return hi.result(cfg, layer, iterations, used, hi.depth-lo.depth)
}

// FixedStepSearch runs uniform layer marching with layers depth intervals.
func FixedStepSearch(field HeightField, ray Ray, layers int) Result {
	cfg := DefaultConfig(ModeFixed)
	cfg.Layers = layers
	cfg.MaxIterations = layers + 1
	return Trace(cfg, field, ray)
}

// BinaryRefineSearch runs fixed-step marching and then binaryIterations
// bisection samples in the crossing bracket.
func BinaryRefineSearch(field HeightField, ray Ray, layers, binaryIterations int) Result {
	cfg := DefaultConfig(ModeBinary)
	cfg.Layers = layers
	cfg.BinaryIterations = binaryIterations
	cfg.MaxIterations = layers + binaryIterations + 1
	return Trace(cfg, field, ray)
}

// MaxIterationSearch bisects the crossing bracket until tolerance is reached
// or maxIterations total height evaluations have been consumed.
func MaxIterationSearch(field HeightField, ray Ray, layers, maxIterations int, tolerance float64) Result {
	cfg := DefaultConfig(ModeMaxIteration)
	cfg.Layers = layers
	cfg.MaxIterations = maxIterations
	cfg.Tolerance = tolerance
	return Trace(cfg, field, ray)
}

// BinarySearch is a descriptive alias for BinaryRefineSearch.
func BinarySearch(field HeightField, ray Ray, layers, binaryIterations int) Result {
	return BinaryRefineSearch(field, ray, layers, binaryIterations)
}

// MaximumIterationSearch is a descriptive alias for MaxIterationSearch.
func MaximumIterationSearch(field HeightField, ray Ray, layers, maxIterations int, tolerance float64) Result {
	return MaxIterationSearch(field, ray, layers, maxIterations, tolerance)
}

// CompareSearches runs all three search strategies with one shared layer count.
// The binary budget and total maximum-iteration budget are supplied separately.
func CompareSearches(field HeightField, ray Ray, layers, binaryIterations, maxIterations int, tolerance float64) []Result {
	fixedCfg := DefaultConfig(ModeFixed)
	fixedCfg.Layers = layers
	fixedCfg.MaxIterations = layers + 1

	binaryCfg := DefaultConfig(ModeBinary)
	binaryCfg.Layers = layers
	binaryCfg.BinaryIterations = binaryIterations
	binaryCfg.MaxIterations = layers + binaryIterations + 1

	maxCfg := DefaultConfig(ModeMaxIteration)
	maxCfg.Layers = layers
	maxCfg.MaxIterations = maxIterations
	maxCfg.Tolerance = tolerance

	return []Result{
		Trace(fixedCfg, field, ray),
		Trace(binaryCfg, field, ray),
		Trace(maxCfg, field, ray),
	}
}
