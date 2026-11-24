package main

type BoundaryResult struct {
	Eye               EyeID     `json:"eye"`
	Bounds            FloatRect `json:"bounds"`
	VisibleRect       FloatRect `json:"visibleRect"`
	Status            string    `json:"status"`
	VisibleAreaRatio  float64   `json:"visibleAreaRatio"`
	MarginLeftPx      float64   `json:"marginLeftPx"`
	MarginRightPx     float64   `json:"marginRightPx"`
	MarginTopPx       float64   `json:"marginTopPx"`
	MarginBottomPx    float64   `json:"marginBottomPx"`
	MinImageMarginPx  float64   `json:"minImageMarginPx"`
	RequiredSafeInset float64   `json:"requiredSafeInsetPx"`
}

func CheckBoundary(bounds FloatRect, eye Eye) BoundaryResult {
	imageRect := eye.ImageRect()
	result := BoundaryResult{
		Eye:               eye.ID,
		Bounds:            bounds,
		RequiredSafeInset: eye.Tracking.SafeInsetPx,
		MarginLeftPx:      bounds.MinX,
		MarginRightPx:     float64(eye.Width) - bounds.MaxX,
		MarginTopPx:       bounds.MinY,
		MarginBottomPx:    float64(eye.Height) - bounds.MaxY,
	}
	result.MinImageMarginPx = minFloat(
		result.MarginLeftPx,
		result.MarginRightPx,
		result.MarginTopPx,
		result.MarginBottomPx,
	)

	if bounds.Width() <= 0.5 || bounds.Height() <= 0.5 || !bounds.Intersects(imageRect) {
		result.Status = "culled"
		result.VisibleAreaRatio = 0
		return result
	}

	visible := bounds.Intersection(imageRect)
	result.VisibleRect = visible
	result.VisibleAreaRatio = 1
	if bounds.Area() > 0 {
		result.VisibleAreaRatio = clampUnit(visible.Area() / bounds.Area())
	}

	if result.VisibleAreaRatio < 0.999 || result.MinImageMarginPx < -0.5 {
		result.Status = "clipped"
		return result
	}

	safe := eye.SafeRect()
	if bounds.MinX < safe.MinX || bounds.MaxX > safe.MaxX ||
		bounds.MinY < safe.MinY || bounds.MaxY > safe.MaxY {
		result.Status = "edge_risk"
		return result
	}
	result.Status = "visible"
	return result
}

func minFloat(values ...float64) float64 {
	out := values[0]
	for _, v := range values[1:] {
		if v < out {
			out = v
		}
	}
	return out
}
