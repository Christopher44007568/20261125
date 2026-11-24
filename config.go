package xrpath

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const (
	FrameCount    = 72
	FrameRateHz   = 60
	PauseAtFrame  = 25
	ReloadAtFrame = 27
	ResumeAtFrame = 29

	FaultPoseFrame       = 14
	FaultResourceFrame   = 22
	FaultVisibilityFrame = 38
	FaultReprojectFrame  = 40
	FaultImageFrame      = 33
	FaultLifecycleFrame  = 26
	FaultParametersFrame = 29
	FaultRelocationFrame = 52
)

func DefaultParameters() Parameters {
	return Parameters{
		Revision:           1,
		Width:              128,
		Height:             80,
		FOVDeg:             68,
		Near:               0.05,
		Far:                24,
		IPD:                0.064,
		Exposure:           1.0,
		ReprojectionPolicy: "depth_aware_auto",
		MaxResourceBytes:   64 << 20,
	}
}

func ReloadedParameters() Parameters {
	p := DefaultParameters()
	p.Revision = 2
	p.FOVDeg = 74
	p.Exposure = 1.12
	p.ReprojectionPolicy = "depth_motion_disocclusion"
	return p
}

func canonicalSHA256(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func bytesSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
