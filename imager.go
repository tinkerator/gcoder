package gcoder

// This file implements Plotters for the g-code sequences. One to
// pre-determine the bounding box for what is being plotted and one to
// render to an image.RGBA. The caller is free to implement their own
// plotter.

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"golang.org/x/image/draw"
	"zappem.net/pub/graphics/raster"
	"zappem.net/pub/math/geom"
)

type Bound struct {
	pts                    int
	oX, oY, lastX, lastY   float64
	minX, maxX, minY, maxY float64
}

func (b *Bound) Command(cmd Command) error {
	switch cmd {
	case CmdSetOrigin:
		b.oX, b.oY = b.lastX, b.lastY
		b.lastX, b.lastY = 0, 0
	default:
		return fmt.Errorf("unsupported command %v", cmd)
	}
	return nil
}

func (b *Bound) MoveTo(x, y, z float64) error {
	b.lastX, b.lastY = b.oX+x, b.oY+y
	if b.pts == 0 || b.lastX < b.minX {
		b.minX = b.lastX
	}
	if b.pts == 0 || b.lastX > b.maxX {
		b.maxX = b.lastX
	}
	if b.pts == 0 || b.lastY < b.minY {
		b.minY = b.lastY
	}
	if b.pts == 0 || b.lastY > b.maxY {
		b.maxY = b.lastY
	}
	b.pts++
	return nil
}

func (b *Bound) LineTo(x, y, z float64) error {
	return b.MoveTo(x, y, z)
}

type plotter struct {
	dy               float64
	sim              *geom.Similarity
	r                *raster.Rasterizer
	oX, oY           float64
	penX, penY, penZ float64
}

func (p *plotter) Command(cmd Command) error {
	switch cmd {
	case CmdSetOrigin:
		p.oX += p.penX
		p.oY += p.penY
		p.penX, p.penY, p.penZ = 0, 0, 0
	default:
		return fmt.Errorf("unsupported command %v", cmd)
	}
	return nil
}

func (p *plotter) MoveTo(x, y, z float64) error {
	px, py := p.sim.Apply(p.oX+x, p.oY+y)
	p.r.MoveTo(px, p.dy-py)
	p.penX, p.penY, p.penZ = x, y, z
	return nil
}

func (p *plotter) LineTo(x, y, z float64) error {
	penx, peny := p.sim.Apply(p.oX+p.penX, p.oY+p.penY)
	px, py := p.sim.Apply(p.oX+x, p.oY+y)
	raster.LineTo(p.r, true, penx, p.dy-peny, px, p.dy-py, 1)
	p.penX, p.penY, p.penZ = x, y, z
	return nil
}

// Make an image.RGBA of size (width,height).
func MakeRGBA(g *Image, width, height int) (*image.RGBA, error) {
	bounds := &Bound{}
	if err := g.Plot(bounds); err != nil {
		log.Fatalf("failed to determine bounds for rendered image: %v", err)
	}
	bounds.minX -= 4
	bounds.maxX += 4
	bounds.minY -= 4
	bounds.maxY += 4
	scale := float64(width) / (bounds.maxX - bounds.minX)
	alt := float64(height) / (bounds.maxY - bounds.minY)
	if scale > alt {
		scale = alt
	}
	plotter := &plotter{
		dy:  float64(height),
		sim: geom.NewSimilarity(0.5*(bounds.minX+bounds.maxX), 0.5*(bounds.minY+bounds.maxY), float64(width)/2, float64(height)/2, scale, 0),
		r:   raster.NewRasterizer(),
	}
	if err := g.Plot(plotter); err != nil {
		return nil, err
	}
	im := image.NewRGBA(image.Rect(0, 0, width, height))
	white := image.NewUniform(color.RGBA{0xff, 0xff, 0xff, 0xff})
	draw.Copy(im, image.Point{}, white, im.Bounds(), draw.Over, nil)
	plotter.r.Render(im, 0, 0, color.RGBA{0xff, 0x00, 0xff, 0xff})
	plotter.r.Reset()

	// Origin at end
	zx, zy := plotter.sim.Apply(plotter.oX, plotter.oY)
	zy = float64(height) - zy
	raster.LineTo(plotter.r, true, zx-4, zy-4, zx+4, zy+4, 1)
	raster.LineTo(plotter.r, true, zx-4, zy+4, zx+4, zy-4, 1)
	plotter.r.Render(im, 0, 0, color.RGBA{0xff, 0x00, 0x00, 0xff})
	plotter.r.Reset()

	// Origin at start
	zx, zy = plotter.sim.Apply(0, 0)
	zy = float64(height) - zy
	raster.LineTo(plotter.r, true, zx-4, zy-4, zx+4, zy+4, 1)
	raster.LineTo(plotter.r, true, zx-4, zy+4, zx+4, zy-4, 1)
	plotter.r.Render(im, 0, 0, color.Black)
	return im, nil
}
