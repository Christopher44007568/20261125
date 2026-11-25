package main

import "math"

type Mode string

const (
	ModeNone      Mode = "none"
	ModeOrdered   Mode = "ordered"
	ModeDiffusion Mode = "diffusion"
)

type MethodInfo struct {
	Mode   Mode
	Name   string
	Short  string
	Kicker string
	Detail string
	File   string
}

var methods = []MethodInfo{
	{
		Mode:   ModeNone,
		Name:   "无抖动",
		Short:  "直接舍入",
		Kicker: "A / DIRECT",
		Detail: "8-bit 舍入，暗部留下清晰的阶梯。",
		File:   "none",
	},
	{
		Mode:   ModeOrdered,
		Name:   "有序抖动",
		Short:  "Bayer 8×8",
		Kicker: "B / ORDERED",
		Detail: "固定阈值矩阵，噪声呈稳定的簇状分布。",
		File:   "ordered",
	},
	{
		Mode:   ModeDiffusion,
		Name:   "误差扩散",
		Short:  "Floyd–Steinberg",
		Kicker: "C / DIFFUSION",
		Detail: "蛇形扫描，把量化误差交给相邻像素。",
		File:   "diffusion",
	},
}

func methodByMode(mode Mode) MethodInfo {
	for _, m := range methods {
		if m.Mode == mode {
			return m
		}
	}
	return methods[0]
}

func bayerMatrix(size int) [][]int {
	matrix := [][]int{{0}}
	for n := 1; n < size; n *= 2 {
		next := make([][]int, n*2)
		for y := range next {
			next[y] = make([]int, n*2)
		}
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				v := matrix[y][x] * 4
				next[y][x] = v
				next[y][x+n] = v + 2
				next[y+n][x] = v + 3
				next[y+n][x+n] = v + 1
			}
		}
		matrix = next
	}
	return matrix
}

var ordered8 = bayerMatrix(8)

func quantize(value float64) float64 {
	if value <= 0 {
		return 0
	}
	if value >= 255 {
		return 255
	}
	return math.Round(value)
}

func orderedChannel(value float64, x, y int) float64 {
	if value <= 0 {
		return 0
	}
	if value >= 255 {
		return 255
	}
	base := math.Floor(value)
	fraction := value - base
	threshold := (float64(ordered8[y%8][x%8]) + 0.5) / 64
	if fraction >= threshold {
		return base + 1
	}
	return base
}

func renderNoDither(s Scene) []uint8 {
	dst := make([]uint8, imageWidth*imageHeight*3)
	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageWidth; x++ {
			c := s.At(x, y)
			i := (y*imageWidth + x) * 3
			dst[i+0] = uint8(quantize(c.R))
			dst[i+1] = uint8(quantize(c.G))
			dst[i+2] = uint8(quantize(c.B))
		}
	}
	return dst
}

func renderOrdered(s Scene) []uint8 {
	dst := make([]uint8, imageWidth*imageHeight*3)
	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageWidth; x++ {
			c := s.At(x, y)
			i := (y*imageWidth + x) * 3
			dst[i+0] = uint8(orderedChannel(c.R, x, y))
			dst[i+1] = uint8(orderedChannel(c.G, x, y))
			dst[i+2] = uint8(orderedChannel(c.B, x, y))
		}
	}
	return dst
}

func renderDiffusion(s Scene) []uint8 {
	pixels := imageWidth * imageHeight
	work := make([][3]float64, pixels)
	dst := make([]uint8, pixels*3)
	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageWidth; x++ {
			c := s.At(x, y)
			work[y*imageWidth+x] = [3]float64{c.R, c.G, c.B}
		}
	}

	for y := 0; y < imageHeight; y++ {
		leftToRight := y%2 == 0
		for step := 0; step < imageWidth; step++ {
			x := step
			if !leftToRight {
				x = imageWidth - 1 - step
			}
			index := y*imageWidth + x
			v := work[index]
			q := [3]float64{quantize(v[0]), quantize(v[1]), quantize(v[2])}
			dst[index*3+0] = uint8(q[0])
			dst[index*3+1] = uint8(q[1])
			dst[index*3+2] = uint8(q[2])
			err := [3]float64{v[0] - q[0], v[1] - q[1], v[2] - q[2]}

			dir := 1
			if !leftToRight {
				dir = -1
			}
			diffuse(work, x+dir, y, 7.0/16.0, err)
			diffuse(work, x-dir, y+1, 3.0/16.0, err)
			diffuse(work, x, y+1, 5.0/16.0, err)
			diffuse(work, x+dir, y+1, 1.0/16.0, err)
		}
	}
	return dst
}

func diffuse(work [][3]float64, x, y int, weight float64, err [3]float64) {
	if x < 0 || x >= imageWidth || y < 0 || y >= imageHeight {
		return
	}
	i := y*imageWidth + x
	work[i][0] += err[0] * weight
	work[i][1] += err[1] * weight
	work[i][2] += err[2] * weight
}
