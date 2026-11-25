# Spectral Resampling Metamerism Report

- Wavelength grid: 380-780 nm at 5 nm steps (81 bands)
- Observer: CIE 1931 2 degree standard observer (trapezoidal spectral integration)
- Reference illuminant: CIE D65 tabulated SPD
- Test illuminant: NARROWBAND LED 450/530/630 NM
- D65 white point XYZ: 95.0414, 100.0000, 108.8725
- LED white point XYZ: 95.7423, 100.0000, 108.2140

## Family-level spectral mismatch

| family | patches | mean dE2000 D65 | max dE2000 D65 | mean dE2000 LED | max dE2000 LED | gain pair split | gain vs spectral | mean spectral RMS |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| GREY | 6 | 0.0000 | 0.0000 | 5.0421 | 11.2666 | 0.0000 | 1.2465 | 0.02342 |
| SKIN | 2 | 0.0000 | 0.0000 | 0.4902 | 0.8175 | 0.0000 | 17.5944 | 0.03322 |
| ORANGE | 2 | 0.0000 | 0.0000 | 0.4012 | 0.4305 | 0.0000 | 24.8628 | 0.02957 |
| OTHER | 14 | 0.0000 | 0.0000 | 1.7854 | 3.8988 | 0.0000 | 38.5460 | 0.02661 |

## Per-patch detail

| # | label | family | dE2000 D65 | dE2000 LED | gain pair split | gain vs spectral | spectral RMS |
|---:|---|---|---:|---:|---:|---:|---:|
| 1 | dark skin | skin | 0.0000 | 0.8175 | 0.0000 | 10.8167 | 0.03332 |
| 2 | light skin | skin | 0.0000 | 0.1629 | 0.0000 | 24.3721 | 0.03311 |
| 3 | blue sky | other | 0.0000 | 3.0279 | 0.0000 | 3.3166 | 0.02698 |
| 4 | foliage | other | 0.0000 | 2.1317 | 0.0000 | 22.0454 | 0.02828 |
| 5 | blue flower | other | 0.0000 | 1.4882 | 0.0000 | 14.6287 | 0.02911 |
| 6 | bluish green | other | 0.0000 | 1.3209 | 0.0000 | 99.9050 | 0.03068 |
| 7 | orange | orange | 0.0000 | 0.4305 | 0.0000 | 31.6702 | 0.03041 |
| 8 | purplish blue | other | 0.0000 | 0.9942 | 0.0000 | 12.6886 | 0.02620 |
| 9 | moderate red | other | 0.0000 | 1.0814 | 0.0000 | 51.0490 | 0.02616 |
| 10 | purple | other | 0.0000 | 2.0288 | 0.0000 | 19.8746 | 0.03234 |
| 11 | yellow green | other | 0.0000 | 0.5757 | 0.0000 | 46.7868 | 0.02937 |
| 12 | orange yellow | orange | 0.0000 | 0.3719 | 0.0000 | 18.0555 | 0.02873 |
| 13 | blue | other | 0.0000 | 3.3995 | 0.0000 | 23.0434 | 0.02865 |
| 14 | green | other | 0.0000 | 3.8988 | 0.0000 | 75.8551 | 0.02583 |
| 15 | red | other | 0.0000 | 1.4100 | 0.0000 | 77.3434 | 0.03015 |
| 16 | yellow | other | 0.0000 | 1.1238 | 0.0000 | 22.0907 | 0.02284 |
| 17 | magenta | other | 0.0000 | 1.4123 | 0.0000 | 59.6154 | 0.02362 |
| 18 | cyan | other | 0.0000 | 1.1022 | 0.0000 | 11.4018 | 0.01233 |
| 19 | white 9.5 (.05 D) | grey | 0.0000 | 0.8003 | 0.0000 | 1.4142 | 0.01397 |
| 20 | neutral 8 (.23 D) | grey | 0.0000 | 1.0937 | 0.0000 | 1.4142 | 0.02347 |
| 21 | neutral 6.5 (.44 D) | grey | 0.0000 | 3.4303 | 0.0000 | 2.2361 | 0.02241 |
| 22 | neutral 5 (.70 D) | grey | 0.0000 | 3.3870 | 0.0000 | 1.4142 | 0.02517 |
| 23 | neutral 3.5 (1.05 D) | grey | 0.0000 | 10.2746 | 0.0000 | 1.0000 | 0.02450 |
| 24 | black 2 (1.5 D) | grey | 0.0000 | 11.2666 | 0.0000 | 0.0000 | 0.03102 |

## Interpretation

The D65 column stays near zero by construction: each metamer pair shares identical
tristimulus values under daylight. The LED column exposes the spectral mismatch that
a single RGB colour-temperature multiplier cannot represent, with separate rows for
grey, skin and orange targets.
