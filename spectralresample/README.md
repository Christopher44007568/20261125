# Spectral Reflectance Resampling and Metamerism Demo (Go)

The same set of reflectance patches under D65 daylight and a narrow-band LED
cannot be handled by multiplying sRGB channels by a colour-temperature gain.
This program works in the wavelength domain instead:

1. resample every measured reflectance spectrum onto a common 5 nm grid
   (380-780 nm, 81 bands);
2. integrate the fixed CIE 1931 2 degree colour matching functions against
   illuminant SPD times reflectance to obtain tristimulus values;
3. build metameric reflectance pairs that are identical under D65 but
   spectrally different, so their failure under the LED is isolated;
4. report the mismatch separately for grey, skin and orange target families.

Only the Go standard library is used. No third-party colour or image
libraries are imported.

## Data

The build-time extractor under `cmd/extractdata` converts upstream
colour-science data snapshots (kept verbatim under `thirdparty/`) into compact
CSV tables in `data/`:

- `cie1931_2deg_5nm.csv` - CIE 1931 2 degree standard observer xbar/ybar/zbar;
- `d65_5nm.csv` - tabulated CIE D65 spectral power distribution;
- `colorchecker_ohta_5nm.csv` - Ohta 1997 ColorChecker Classic 24 patch
  reflectances (dark/light skin, orange, orange yellow, six neutrals, ...).

Re-run extraction with:

```
go run ./cmd/extractdata
```

## Method

### Spectral integration

Tristimulus values use trapezoidal integration over the shared grid:

```
X = 100 * integral(S(lambda) R(lambda) xbar(lambda)) /
             integral(S(lambda) ybar(lambda))
```

and analogously for Y and Z, so a perfect white reflector gives Y = 100 under
any illuminant. LED stimuli are Bradford-adapted from the LED white point to
the D65 white point before CIE L*a*b* and CIEDE2000 evaluation, which
separates reflectance mismatch from the illuminant white shift.

### Narrow-band LED

The LED is a sum of three Gaussian emitters at 450, 530 and 630 nm
(FWHM 20-22 nm). Emitter powers are balanced so the LED white point
(95.74, 100, 108.21) is close to D65 (95.04, 100, 108.87); remaining
differences therefore come from spectral shape, not white balance.

### Metamer construction

For each patch a perturbation is built in the null space of the weighted CMF
constraint matrix `B = diag(S_D65) [xbar ybar zbar]^T` using a hand-written
Householder QR decomposition. Perturbations are smooth sinusoids (2-7 cycles
across the visible range) projected into `ker(B)`, then alternated with
[0, 1] reflectance clipping for a fixed point. The result keeps D65
tristimulus values matched within 0.05 while producing a physically plausible
different spectrum.

### Why RGB gain fails

A single per-channel gain is calibrated on the white point and applied to the
D65-rendered sRGB values. Because a gain is rank-one per channel it cannot
distinguish two spectra with identical D65 tristimulus values: the metamer
pair split is exactly zero. The full spectral pipeline shows CIEDE2000 errors
up to roughly 11 under the LED for the same patches, while the gain shortcut
diverges from the true LED rendering by up to ~100 RGB units on saturated
targets (see `errormap_rgb_gain.ppm`).

## Build and run

```
go test ./...
go run ./cmd/spectraldemo -data data -out output
```

## Outputs

All raster images are binary PPM (P6), written by the package itself;
equivalent SVG line charts are emitted as well.

- `patches_d65_base.ppm`, `patches_d65_metamer.ppm` - D65 swatch charts;
- `patches_led_base.ppm`, `patches_led_metamer.ppm` - LED swatch charts;
- `patches_rgb_gain_prediction.ppm` - what a white-point RGB gain predicts;
- `errormap_d65.ppm` - D65 CIEDE2000 heatmap (stays at zero by construction);
- `errormap_led.ppm` - LED CIEDE2000 heatmap exposing spectral mismatch;
- `errormap_rgb_gain.ppm` - RGB distance between gain shortcut and spectral LED;
- `illuminants_spd.ppm/.svg` - D65 versus narrow-band LED SPD;
- `spectra_grey.ppm/.svg`, `spectra_skin.ppm/.svg`, `spectra_orange.ppm/.svg`
  - solid base versus dashed metamer reflectance per inspected family;
- `report.csv`, `report.md` - per-patch XYZ, dE2000 and family statistics.

## Package layout

- `spectrum.go` - CSV loading, interpolation and multi-band resampling;
- `colorimetry.go` - CMF, illuminants, trapezoidal tristimulus integration;
- `colorspace.go` - sRGB, CIE Lab, Bradford adaptation, CIEDE2000;
- `linalg.go` - Householder QR and null-space solver;
- `metamer.go` - smooth null-space metamer construction;
- `ppm.go`, `render.go`, `rasterplot.go`, `svg.go` - dependency-free output;
- `pipeline.go` - orchestration and the naive RGB-gain comparison;
- `cmd/spectraldemo` - CLI producing every artefact;
- `cmd/extractdata` - build-time CSV extractor for the reference data.
