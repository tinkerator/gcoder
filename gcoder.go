// Package gcoder builds up g-code step sequences, referred to as an
// Image. The package includes native support for the Snapmaker A350
// 3-in-1 system (Laser, CNC, 3D printer).
package gcoder

import (
	"errors"
	"fmt"
)

type Command int

const (
	CmdInvalid Command = iota
	CmdSetOrigin
)

// Step holds an atom of g-code work.
type Step struct {
	X, Y, Z float64
	Rel     bool
	Active  bool
	Power   float64
	Speed   int
	Comment string
	Command Command
}

// String serializes a (*Step) value.
func (s *Step) String() string {
	return fmt.Sprintf("%#v", *s)
}

// Image holds a series of (*Step) values.
type Image struct {
	Steps  []*Step
	Copies int
}

// Start a new, empty, Image.
func NewImage() *Image {
	return &Image{
		Copies: 1,
	}
}

// ErrInvalidTool etc are the error return values for the gcoder
// package.
var (
	ErrInvalidTool  = errors.New("invalid tool selection")
	ErrInvalidPower = errors.New("invalid tool power")
)

// Note adds a comment to the image stream.
func (im *Image) Note(note string) {
	im.Steps = append(im.Steps, &Step{
		Comment: note,
	})
}

// SetSpeed sets the tool n speed (in mm/minute).
func (im *Image) SetSpeed(active bool, speed int) {
	im.Steps = append(im.Steps, &Step{
		Speed:  speed,
		Rel:    true,
		Active: active,
	})
}

// moveXY initializes a *Step with a destination (x,y).
func moveXY(x, y float64) *Step {
	return &Step{
		X: x,
		Y: y,
	}
}

// MoveXY relocates the tool head to a specific location. This move is
// accomplished without the tool being active.
func (im *Image) MoveXY(x, y float64) error {
	im.Steps = append(im.Steps, moveXY(x, y))
	return nil
}

// Cut a line from the current location to (x, y) with power level
// [0,100].
func (im *Image) LineXY(x, y, power float64) error {
	if power < 0 || power > 100 {
		return ErrInvalidPower
	}
	s := moveXY(x, y)
	s.Active = true
	s.Power = power
	im.Steps = append(im.Steps, s)
	return nil
}

// Raise increases the relative Z value without any line drawn.
func (im *Image) Raise(dz float64) error {
	if dz == 0 {
		return nil
	}
	im.Steps = append(im.Steps, &Step{
		Z:   dz,
		Rel: true,
	})
	return nil
}

// Drill increases the relative Z value while drawing. To lower while
// drilling provide a negative value for dz.
func (im *Image) Drill(dz float64, power float64) error {
	if dz == 0 {
		return nil
	}
	im.Steps = append(im.Steps, &Step{
		Z:      dz,
		Power:  power,
		Active: true,
		Rel:    true,
	})
	return nil
}

// SetOrigin resets the coordinate system to have the current
// location become the new (0,0,0) coordinate.
func (im *Image) SetOrigin() error {
	im.Steps = append(im.Steps, &Step{
		Command: CmdSetOrigin,
	})
	return nil
}

// Plotter is an interface to a 3D pen plotter.
type Plotter interface {
	Command(cmd Command) error
	MoveTo(x, y, z float64) error
	LineTo(x, y, z float64) error
}

// Plot executes the Image using the provided Plotter.
func (im *Image) Plot(plotter Plotter) error {
	var penX, penY, penZ float64
	for _, s := range im.Steps {
		if s.Comment != "" {
			continue // ignore
		}
		if s.Command != CmdInvalid {
			if err := plotter.Command(s.Command); err != nil {
				return err
			}
			continue
		}
		if s.Rel {
			penX = penX + s.X
			penY = penY + s.Y
			penZ = penZ + s.Z // Only changed in relative movement.
			if err := plotter.MoveTo(penX, penY, penZ); err != nil {
				return err
			}
			continue
		}
		penX, penY = s.X, s.Y
		if s.Active {
			if err := plotter.LineTo(penX, penY, penZ); err != nil {
				return err
			}
			continue
		}
		if err := plotter.MoveTo(penX, penY, penZ); err != nil {
			return err
		}
	}
	return nil
}
