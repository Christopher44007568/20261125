package main

import "log"

func main() {
	f := buildFeather()

	top, _ := topView(f)
	if err := savePNG("artifacts/transparent.png", layerToRGBA(top)); err != nil {
		log.Fatal(err)
	}
	if err := savePNG("artifacts/alpha_mask.png", alphaToGray(featherOnlyView(f))); err != nil {
		log.Fatal(err)
	}
	topBG := checkerBackground(top.W, top.H, 28)
	if err := savePNG("artifacts/top.png", composite(topBG, top)); err != nil {
		log.Fatal(err)
	}

	ob := obliqueView(featherOnlyView(f), f)
	obBG := checkerBackground(ob.W, ob.H, 28)
	if err := savePNG("artifacts/oblique.png", composite(obBG, ob)); err != nil {
		log.Fatal(err)
	}

	cu := closeupView(f)
	cuBG := checkerBackground(cu.W, cu.H, 28)
	if err := savePNG("artifacts/closeup.png", composite(cuBG, cu)); err != nil {
		log.Fatal(err)
	}
}
