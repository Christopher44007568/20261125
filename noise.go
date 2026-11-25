package main

import "math"

func hash3(x, y, z int) uint32 {
	h := uint32(x)*0x8da6b343 ^ uint32(y)*0xd8163841 ^ uint32(z)*0xcb1ab31f
	h ^= h >> 16
	h *= 0x7feb352d
	h ^= h >> 15
	h *= 0x846ca68b
	h ^= h >> 16
	return h
}

func hash2(x, y int) uint32 { return hash3(x, y, 0x51ed270b) }

func rand01(x, y int) float64 {
	return float64(hash2(x, y)&0x00ffffff) / float64(0x01000000)
}

func rand013(x, y, z int) float64 {
	return float64(hash3(x, y, z)&0x00ffffff) / float64(0x01000000)
}

func fade(t float64) float64 { return t * t * t * (t*(t*6-15) + 10) }

func value3(x, y, z float64) float64 {
	xi, yi, zi := int(math.Floor(x)), int(math.Floor(y)), int(math.Floor(z))
	tx, ty, tz := fade(x-float64(xi)), fade(y-float64(yi)), fade(z-float64(zi))
	a := mix(rand013(xi, yi, zi), rand013(xi+1, yi, zi), tx)
	b := mix(rand013(xi, yi+1, zi), rand013(xi+1, yi+1, zi), tx)
	c := mix(rand013(xi, yi, zi+1), rand013(xi+1, yi, zi+1), tx)
	d := mix(rand013(xi, yi+1, zi+1), rand013(xi+1, yi+1, zi+1), tx)
	lo := mix(a, b, ty)
	hi := mix(c, d, ty)
	return mix(lo, hi, tz)
}

func value2(x, y float64) float64 {
	xi, yi := int(math.Floor(x)), int(math.Floor(y))
	tx, ty := fade(x-float64(xi)), fade(y-float64(yi))
	a := mix(rand01(xi, yi), rand01(xi+1, yi), tx)
	b := mix(rand01(xi, yi+1), rand01(xi+1, yi+1), tx)
	return mix(a, b, ty)
}

func fbm2(x, y float64, octaves int) float64 {
	sum, amp, norm := 0.0, 0.5, 0.0
	for i := 0; i < octaves; i++ {
		sum += amp * value2(x, y)
		norm += amp
		x = x*2.03 + 17.1
		y = y*2.03 + 9.2
		amp *= 0.5
	}
	return sum / norm
}

func fbm3(p Vec3, freq float64, octaves int) float64 {
	sum, amp, norm := 0.0, 0.5, 0.0
	q := p.Mul(freq)
	for i := 0; i < octaves; i++ {
		sum += amp * value3(q.X, q.Y, q.Z)
		norm += amp
		q = q.Mul(2.03).Add(V(17.1, 9.2, 31.7))
		amp *= 0.5
	}
	return sum / norm
}
