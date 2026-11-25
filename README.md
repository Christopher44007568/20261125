# Long hair through glass

Go standard-library renderer for a fast head turn in which long hair crosses
several semi-transparent glass blades. The renderer uses a per-pixel deep
sample stack instead of one transparent card per hair bundle.

## Run

```sh
go run .
```

The command writes these files to `artifacts/`:

- `color.png`: final sRGB frame with spatial supersampling and a 0.060 s
  temporal shutter.
- `layer_order.png`: transparent-layer order mask. The palette runs from
  violet (back) through cyan/green/yellow/orange to pink (front); opaque head
  and clothing surfaces are neutral gray.
- `motion_vectors.png`: optical-flow-style visualization. Hue is direction,
  value and arrow length are speed, and arrows are placed on moving hair.
- `metadata.json`: counts, sort key, shutter settings, and encoding details.

Optional flags:

```sh
go run . -width 1600 -height 1000 -ss 2 -samples 24 -shutter 0.06 -out artifacts
```

## Rendering model

- Every one of the 150 strands is a separately rasterized, tapered curve with
  its own depth, alpha, color, and motion samples.
- Each output pixel gathers hair strands, glass blades, and opaque geometry,
  then sorts back to front by `(depth, sequence, kind, layer id)` with a stable
  sort. Glass is therefore free to sit between individual hair strands.
- Hair motion is evaluated per curve parameter, not per bundle. During a turn,
  roots follow the head while tips lag and whip with a stronger inertia term.
- Each shutter sample re-deforms every strand from root to tip. Their ghost
  trails are composited with the same depth rules, so the tip trail follows
  the tip's own velocity field.
- Depth is sampled at the nearest stamp on each curve; intersecting stamps from
  the same strand are merged instead of accumulating extra opacity.

## Verify

```sh
go test ./...
go vet ./...
```

The tests cover stable equal-depth ordering, front-to-back alpha compositing,
per-strand tip travel, and deterministic output.
