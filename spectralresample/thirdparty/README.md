# Upstream reference snapshots

These Python data files are copied verbatim from the BSD-3-licensed
[colour-science/colour](https://github.com/colour-science/colour) repository
and are only parsed by the build-time extractor in `cmd/extractdata`. The
runtime Go program never reads them; it uses the generated CSV tables in
`data/`.

- `cmfs.py` - CIE standard observer colour matching functions, including the
  CIE 1931 2 degree observer at 1 nm spacing;
- `d65_sds.py` - tabulated CIE D65 spectral power distribution (5 nm);
- `colorchecker_sds.py` - Ohta 1997 ColorChecker Classic 24 patch
  reflectance spectra (380-730 nm, 10 nm) and BabelColor average data.

Re-fetch with:

```
curl -L -o cmfs.py \
  https://raw.githubusercontent.com/colour-science/colour/develop/colour/colorimetry/datasets/cmfs.py
curl -L -o d65_sds.py \
  https://raw.githubusercontent.com/colour-science/colour/v0.3.16/colour/colorimetry/datasets/illuminants/sds.py
curl -L -o colorchecker_sds.py \
  https://raw.githubusercontent.com/colour-science/colour/develop/colour/characterisation/datasets/colour_checkers/sds.py
```
