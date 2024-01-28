package main

import (
	"errors"
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	ts "github.com/tinyspline/go"
)

type Sp struct {
	curve  ts.BSpline
	degree int
}

func GetPointAt(spline ts.BSpline, u float64) ([]float64, []float64) {
	net := spline.Eval(u)
	pts := net.GetPoints()
	res := net.GetResult()
	return res, pts
}
func (s *Sp) NewSpline(pts *[]float64) error {
	if len(*pts)/2 < s.degree+1 {
		return errors.New("less than enough ctrl points")
	}
	s.curve = ts.NewBSpline(len(*pts)/2, 2, s.degree)
	s.curve.SetControlPoints(*pts)
	return nil
}
func DrawSpline(s *Sp, num int, ops *op.Ops) {
	if s == nil || s.curve == nil {
		return
	}
	ctrlPts := s.curve.GetControlPoints()
	pts := s.curve.Sample(num * len(ctrlPts) / 2)
	dim := s.curve.GetDimension()

	p := DrawLine(&pts, dim, ops)
	c := state.Color()
	paint.FillShape(ops, c, clip.Stroke{Path: p.End(), Width: 4}.Op())
}

func DrawLine(pts *[]float64, dim int, ops *op.Ops) clip.Path {
	pts1 := *pts
	n := make([]struct{}, int(len(pts1)/dim-1))
	var p clip.Path
	p.Begin(ops)
	for k := range n {
		p0x := pts1[k*dim]
		p0y := pts1[k*dim+1]
		p1x := pts1[(k+1)*dim]
		p1y := pts1[(k+1)*dim+1]
		if k == 0 {
			p.MoveTo(f32.Pt(float32(p0x), float32(p0y)))
		}
		p.LineTo(f32.Pt(float32(p1x), float32(p1y)))
	}
	return p
}
func DrawPoints(pts *[]float64, ops *op.Ops, c color.NRGBA) {
	for k := range *pts {
		if k%2 == 1 {
			continue
		}
		x, y := (*pts)[k], (*pts)[k+1]
		circle := clip.Ellipse{Min: image.Pt(int(x)-4, int(y)-4), Max: image.Pt(int(x)+4, int(y)+4)}
		paint.FillShape(ops, c, circle.Op(ops))
	}
}
