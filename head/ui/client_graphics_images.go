//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"image"
	"image/draw"
	_ "image/jpeg" // register the JPEG decoder for image.Decode
	_ "image/png"  // register the PNG decoder for image.Decode (the primary overlay format)
	"os"

	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/clientgraphics"
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
)

// Image billboards (M16-F05 #641) are world-anchored images an add-in overlays in the model
// space. Like text labels they are 2D: the host loads the image file into a GPU texture once,
// then each frame projects the anchor and blits the texture at its pixel — no 3D pipeline.

// gfxImageWin is the window the billboard textures are created on, bound each frame (the texture
// handles belong to that window); gfxImageTex caches one texture per loaded path (0 ⇒ a load
// that failed, so it is not retried every frame).
var (
	gfxImageWin *native.Window
	gfxImageTex = map[string]uint64{}
)

// bindGraphicsImages points the billboard texture cache at the current window.
func bindGraphicsImages(win *native.Window) { gfxImageWin = win }

// drawClientGraphicsImages projects each image billboard's world anchor and blits its texture,
// centered on the anchor and sized from the billboard's model dimensions. Anchors behind the
// camera or images that fail to load are skipped.
func drawClientGraphicsImages(cx, cy float32, cam scene.Camera, images []clientgraphics.ImageBillboard) {
	wpp := cam.WorldPerPixel()
	for _, im := range images {
		tex, ok := gfxTexture(im.Path)
		if !ok {
			continue
		}
		sx, sy, vis := renderer.Project(cam, viewportNear, viewportFar, im.Anchor)
		if !vis {
			continue
		}
		w, h := billboardPixels(im, wpp)
		native.SetCursorPos(cx+float32(sx)-w/2, cy+float32(sy)-h/2)
		native.Image(tex, w, h)
	}
}

// billboardPixels converts a billboard's model dimensions to screen pixels (a sensible default
// when unset), clamped so a degenerate size never produces a zero/huge quad.
func billboardPixels(im clientgraphics.ImageBillboard, wpp float64) (w, h float32) {
	w, h = float32(im.Width/wpp), float32(im.Height/wpp)
	if w < 1 {
		w = 64
	}
	if h < 1 {
		h = 64
	}
	return w, h
}

// gfxTexture returns the cached texture for path, loading it on first use (0/false on failure,
// remembered so a bad path is not re-read every frame).
func gfxTexture(path string) (uint64, bool) {
	if tex, ok := gfxImageTex[path]; ok {
		return tex, tex != 0
	}
	if gfxImageWin == nil {
		return 0, false
	}
	rgba, w, h, err := decodeImageRGBA(path)
	if err != nil {
		gfxImageTex[path] = 0 // remember the failure
		return 0, false
	}
	tex := gfxImageWin.CreateTexture(rgba, w, h)
	gfxImageTex[path] = tex
	return tex, true
}

// decodeImageRGBA reads an image file and returns its tightly-packed RGBA bytes and size.
func decodeImageRGBA(path string) (pix []byte, w, h int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, 0, 0, err
	}
	b := img.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	return rgba.Pix, b.Dx(), b.Dy(), nil
}
