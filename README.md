# 20261125

Repository initialized for 20261125.

## Procedural Feather

Pure Go (standard library only) programmatic feather renderer.

Run:

```sh
go run .
```

Outputs in `artifacts/`:

- `top.png`: top view over a checker and solid-color background, with wind arrow and fixed anchors.
- `oblique.png`: perspective-warped oblique view; alpha is warped in premultiplied space.
- `closeup.png`: enlarged barb field showing tangent arrows, anchor rings, and coverage weights.
- `transparent.png`: standalone straight-alpha feather for external compositing.
- `alpha_mask.png`: feather-only coverage mask.

Invariants:

- The rachis is a fixed cubic curve and never moves under wind.
- Each barb starts exactly at a rachis anchor; only its control point and tip receive wind displacement.
- All strokes use coverage-weighted premultiplied alpha, so soft edges composite correctly over arbitrary backgrounds.

Tests:

```sh
go test ./...
```
