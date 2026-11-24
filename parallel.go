package main

import (
	"runtime"
	"sync"
)

func parallelRows(height int, fn func(y int)) {
	workers := runtime.GOMAXPROCS(0)
	if workers > height {
		workers = height
	}
	if workers <= 1 {
		for y := 0; y < height; y++ {
			fn(y)
		}
		return
	}
	var wg sync.WaitGroup
	wg.Add(workers)
	for worker := 0; worker < workers; worker++ {
		go func(start int) {
			defer wg.Done()
			for y := start; y < height; y += workers {
				fn(y)
			}
		}(worker)
	}
	wg.Wait()
}
