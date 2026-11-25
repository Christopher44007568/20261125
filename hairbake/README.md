# hairbake — strand groom to hair-card bakery (Go stdlib only)

Bakes a strand groom into a small set of hair cards and produces audit
renderings comparing the reduced representation against the original strands.
Everything (groom generation, k-means clustering, atlas rasterization,
software 3D rendering, PNG encoding, bitmap/CJK labeling) uses only the Go
standard library.

## Run

```
go run ./hairbake [output-dir]   # default output-dir: out
```

## Pipeline

1. `groom.go` — deterministic procedural groom: 680 strands x 24 segments on
   an ellipsoid scalp, including ~15% noisy flyaway strands.
2. `bake.go` — k-means++ clustering of strand roots into 10 ribbon cards.
   Ribbon widths extend to the outermost strand with overlap, taper at the
   tip, and upper knots are lifted onto the scalp so silhouette coverage and
   specular continuity take priority over interior detail.
3. `atlas.go` — packs cards into a 2x5, 1024x1024 island grid with empty
   gutters; rasterizes color+coverage (`atlas_color.png`) and tangent flow +
   specular mask (`atlas_flow_spec.png`).
4. `render*.go` / `raster.go` — pinhole camera, ellipsoid ray casting,
   depth-sorted strand rendering, z-buffered card rasterization and
   Kajiya-Kay style anisotropic shading.
5. `annotate.go` / `composite.go` / `closeup.go` / `montage.go` — side-by-
   side audit, zoomed closeup, five-view puzzle and bilingual callouts:
   red rings mark lost flyaway tips, yellow dashed lines mark texture seams.

## Outputs (in the output directory)

- `atlas_color.png`, `atlas_flow_spec.png`, `atlas_annotated.png`
- `side_by_side.png` — front-view reference vs baked cards
- `closeup.png` — 2x zoomed silhouette/flyaway comparison
- `montage_views.png` — front/left/right/back/top, reference over cards
- `render_reference_front.png`, `render_cards_front.png`
- `summary.json`, `REPORT.md` — reduction and quality metrics

## Design priorities under the card budget

Silhouette coverage first (ribbon overlap + scalp-hugging roots), highlight
retention second (flow/spec map drives per-pixel anisotropic shading); sparse
flyaway arcs that no ribbon can represent are flagged rather than faked.
