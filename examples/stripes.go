// Program stripes generates some g-code for the A350 1.6W laser
// to scribe some different width stripes. These can be used to
// investigate laser focus, speed and intensity.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"

	"zappem.net/pub/graphics/polymark"
	"zappem.net/pub/io/gcoder"
	"zappem.net/pub/math/polygon"
)

var (
	dest      = flag.String("dest", "stripes.nc", "destination laser file")
	speed     = flag.Int("speed", 3000, "motion speed with laser (mm/min)")
	fly       = flag.Int("fly", 3000, "motion speed without laser (mm/min)")
	power     = flag.Float64("power", 80, "laser %power [0..100]")
	fill      = flag.Bool("fill", true, "raster inside the polygons")
	copies    = flag.Int("copies", 1, "number of times to run over pattern")
	width     = flag.Int("width", 720, "width of PNG image in NC file")
	height    = flag.Int("height", 480, "height of PNG image in NC file")
	laserTool = flag.String("laser", "1.6", "wattage of laser tool head")
)

func createNCFile(name string, g *gcoder.Image) error {
	im, err := gcoder.MakeRGBA(g, *width, *height)
	p := &bytes.Buffer{}
	if err := png.Encode(p, im); err != nil {
		return fmt.Errorf("failed to encode PNG (for %q): %v", name, err)
	}
	if err := os.WriteFile(name+".png", p.Bytes(), 0777); err != nil {

		return fmt.Errorf("failed to write a PNG file %q.png: %v", name, err)
	}
	f, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("failed to create %q: %v", name, err)
	}
	defer f.Close()
	laser, ok := gcoder.LaserWattage[*laserTool]
	if !ok {
		return fmt.Errorf("%q is not a recognized laser tool head", *laserTool)
	}
	if err := g.A350Laser(laser, f, p.Bytes()); err != nil {
		return fmt.Errorf("failed to write laser code to %q: %v", name, err)
	}
	return nil
}

func main() {
	flag.Parse()

	laser, ok := gcoder.LaserWattage[*laserTool]
	if !ok {
		log.Fatalf("unrecognized laser %q", *laserTool)
	}
	beam := gcoder.LaserWidth[laser] / 2

	var polys *polygon.Shapes
	from := 0.0
	var notes []string
	for w := 0.1; from <= 2; w += 0.1 {
		from += w / 2
		pts := []polygon.Point{
			{0, from},
			{5 - from, from},
			{10 - from, 5 + from},
			{10 - from, 10 + from},
		}
		pen := &polymark.Pen{
			Scribe:  0.1,
			Reflect: true,
		}
		notes = append(notes, fmt.Sprintf("line width %.2fmm", w))
		polys = pen.Line(polys, pts, w, true, true)
		from += w * 3 / 2
	}
	polys.Union()

	g := gcoder.NewImage()
	g.Copies = *copies
	g.SetSpeed(false, *fly)
	g.SetSpeed(true, *speed)

	var holes []int
	for i, line := range polys.P {
		if line.Hole {
			holes = append(holes, i)
		}
		start := line.PS[len(line.PS)-1]
		g.Note(notes[i])
		g.MoveXY(start.X, start.Y)
		for _, pt := range line.PS {
			g.LineXY(pt.X, pt.Y, *power)
		}
	}

	if *fill {
		for i, s := range polys.P {
			if s.Hole {
				continue
			}
			g.Note(notes[i])
			lines, err := polys.Slice(i, beam, holes...)
			if err != nil {
				log.Fatalf("slice %v: %d", err, len(lines))
			}
			vlines, err := polys.VSlice(i, beam, holes...)
			if err != nil {
				log.Fatalf("vslice %v: %d", err, len(lines))
			}
			lines = append(lines, vlines...)
			polygon.OptimizeLines(lines)
			for _, line := range lines {
				g.MoveXY(line.From.X, line.From.Y)
				g.LineXY(line.To.X, line.To.Y, *power)
			}
		}
	}

	g.MoveXY(0, 0)

	if err := createNCFile(*dest, g); err != nil {
		log.Fatalf("%q file creation failed: %v", err)
	}
	log.Printf("generated %q and %q", *dest, *dest+".png")
}
