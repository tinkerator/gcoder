// Program cncpoly carves a polygon with the current CNC bit.  The
// program expects to start at the origin point and will grind out a
// depth of cut exactly on the line. All holes will be carved
// first. Then all regular (non-hole) polygons.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/png"
	"log"
	"math"
	"os"
	"time"

	"zappem.net/pub/io/gcoder"
	"zappem.net/pub/math/polygon"
)

var (
	targetX = flag.Float64("x", 0, "displaced origin (0,?) of work polygon")
	targetY = flag.Float64("y", 0, "displaced origin (?,0) of work polygon")
	theta   = flag.Float64("theta", 0, "counter-clockwise rotation (deg) of polygon (around its 0,0)")
	hover   = flag.Float64("hover", 1.0, "height over surface to fly CNC bit")
	depth   = flag.Float64("depth", 3.0, "depth of cut below z-origin")
	step    = flag.Float64("step", 0.25, "each layer slice depth delta (+ve)")
	poly    = flag.String("poly", "", "polygon.Shapes json file to render")
	dest    = flag.String("dest", "", "name of generated .cnc file")
	speed   = flag.Int("speed", 100, "cutting speed for router")
	fly     = flag.Int("fly", 3000, "fly-over speed for router")
	scale   = flag.Float64("scale", 1.0, "magnification for polygon")
	power   = flag.Float64("power", 100.0, "power percentage for cut")
	width   = flag.Int("width", 720, "width of PNG image")
	height  = flag.Int("height", 480, "height of PNG image")
)

func createCNCFile(name string, g *gcoder.Image) error {
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
	if err := g.A350CNC(f, p.Bytes()); err != nil {
		return fmt.Errorf("failed to write CNC code to %q: %v", name, err)
	}
	return nil
}

func main() {
	flag.Parse()

	if *poly == "" {
		log.Fatalf("mandatory --poly=<file.json> flag required")
	}
	p, err := polygon.JSONFromFile(*poly)
	if err != nil {
		log.Fatalf("failed to read %q: %v", *poly, err)
	}

	p = p.Transform(polygon.Point{}, polygon.Point{*targetX, *targetY}, math.Pi/180**theta, *scale)

	g := gcoder.NewImage()
	g.SetSpeed(false, *fly)
	g.SetSpeed(true, *speed)

	g.Raise(*hover)
	g.SetSpeed(true, *fly)

	for down := *step; down <= *depth; down += *step {
		for _, line := range p.P {
			if !line.Hole {
				continue
			}
			start := line.PS[len(line.PS)-1]
			g.MoveXY(start.X, start.Y)
			g.SetSpeed(true, *speed)
			depth := *hover + down
			g.Drill(-depth, 100)
			g.Wait(5 * time.Millisecond)
			for _, pt := range line.PS {
				g.LineXY(pt.X, pt.Y, *power)
			}
			g.Drill(depth, 100)
			g.SetSpeed(true, *fly)
		}

		for _, line := range p.P {
			if line.Hole {
				continue
			}
			start := line.PS[len(line.PS)-1]
			g.MoveXY(start.X, start.Y)
			g.SetSpeed(true, *speed)
			depth := *hover + down
			g.Drill(-depth, 100)
			g.Wait(5 * time.Millisecond)
			for _, pt := range line.PS {
				g.LineXY(pt.X, pt.Y, *power)
			}
			g.Drill(depth, 100)
			g.SetSpeed(true, *fly)
		}
	}
	g.MoveXY(0, 0)

	if err := createCNCFile(*dest, g); err != nil {
		log.Fatalf("failed to create %q: %v", *dest, err)
	}
}
