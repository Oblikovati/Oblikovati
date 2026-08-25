//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Command openpbrgoldenshot is the "our side" of M45-F05 PBI-353's Blender perceptual
// golden harness (architecture/testing/00-renderer-oracle-pipeline.md, Tier 4): renders
// a unit UV sphere, one directional light, and an OpenPBR material through the live
// path tracer (the exact per-pixel dispatch renderer.Realistic mode uses,
// head/internal/native's RTScene/SWScene), tone-maps through
// kernel/shading/openpbr.ToDisplay, and writes a PNG — matching
// test-utilities/openpbr-goldens/oracle/render_reference.py's scene parameters exactly
// (same params.json shape) so the two images are comparable.
//
//	go run ./head/cmd/openpbrgoldenshot -params params.json -out ours.png
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image/png"
	"os"

	"oblikovati.org/head/internal/native"
)

// goldenParams mirrors render_reference.py's params.json exactly — only the fields the
// live shader's base_lobes.glsl can consume are used (see this file's own doc comment).
type goldenParams struct {
	BaseColor      [3]float32 `json:"base_color"`
	Metallic       float32    `json:"metallic"`
	Roughness      float32    `json:"roughness"`
	IOR            float32    `json:"ior"`
	LightDir       [3]float32 `json:"light_direction"`
	LightIntensity float32    `json:"light_intensity"`
	Eye            [3]float32 `json:"eye"`
	TanHalfFovY    float32    `json:"tan_half_fov_y"`
	Width          int        `json:"width"`
	Height         int        `json:"height"`
}

func main() {
	paramsPath := flag.String("params", "", "path to a golden params.json (render_reference.py's shape)")
	outPath := flag.String("out", "", "output PNG path")
	software := flag.Bool("software", false, "force the software (compute-BVH) backend instead of hardware RT")
	flag.Parse()
	if *paramsPath == "" || *outPath == "" {
		fmt.Fprintln(os.Stderr, "usage: openpbrgoldenshot -params params.json -out ours.png")
		os.Exit(2)
	}
	if err := run(*paramsPath, *outPath, *software); err != nil {
		fmt.Fprintln(os.Stderr, "openpbrgoldenshot:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "wrote", *outPath)
}

func run(paramsPath, outPath string, forceSoftware bool) error {
	data, err := os.ReadFile(paramsPath)
	if err != nil {
		return fmt.Errorf("read params: %w", err)
	}
	p := goldenParams{Width: 256, Height: 256} // defaults, overridden by json if present
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("parse params: %w", err)
	}
	if p.IOR == 0 {
		p.IOR = 1.5 // IOR=0 is never physically valid, so it's a safe "unset in JSON" sentinel
	}

	win, err := native.CreateWindow(64, 64, "openpbrgoldenshot")
	if err != nil {
		return fmt.Errorf("create window: %w", err)
	}
	defer win.Destroy()

	triangles := native.UVSphereTriangles(1, 96, 48, 1)
	basis := native.PinholeCameraBasis(p.Eye, p.TanHalfFovY, p.Width, p.Height)
	light := native.RealisticLightParams{
		LightDirection: p.LightDir, LightIntensity: p.LightIntensity, LightColor: [3]float32{1, 1, 1},
		BaseColor: p.BaseColor, BaseWeight: 1, SpecularRoughness: p.Roughness, SpecularIOR: p.IOR, BaseMetalness: p.Metallic,
	}
	pixels, err := native.RenderGoldenSphere(win, triangles, basis, light, p.Width, p.Height, forceSoftware)
	if err != nil {
		return err
	}
	return writePNG(outPath, pixels, p.Width, p.Height)
}

func writePNG(path string, pixels []float32, w, h int) error {
	img := native.ToneMappedImage(pixels, w, h)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
