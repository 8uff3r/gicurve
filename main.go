package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"runtime/pprof"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	ts "github.com/tinyspline/go"
)

const (
	screenWidth  = 960
	screenHeight = 1000
)

type App struct{}

var pressed = false

func main() {
	// go func() {
	// 	http.ListenAndServe("localhost:8080", nil)
	// }()
	// Start CPU profiling
	f, err := os.Create("cpu.pprof")
	if err != nil {
		panic(err)
	}
	pprof.StartCPUProfile(f)
	func() {

		window := app.NewWindow(app.MaxSize(unit.Dp(screenWidth), unit.Dp(screenHeight)))
		err := draw(window)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StopCPUProfile()
		os.Exit(0)
	}()
	app.Main()
}

var pts []float64
var spline *Sp
var dragPt int

func draw(window *app.Window) error {

	spline = &Sp{degree: 3}

	var ops op.Ops
	// a1 := 0
	// a2 := 1
	for {
		switch e := window.NextEvent().(type) {
		case system.DestroyEvent:
			return e.Err
		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)

			layout.Flex{Axis: layout.Horizontal}.Layout(
				gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// w, h := 900, 800
					// drawScene(gtx.Ops, gtx.Queue, a1, w, h)
					return layout.Dimensions{Size: image.Pt(screenWidth*.8, screenHeight)}
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// op.Offset(image.Pt(gtx.Dp(200), gtx.Dp(150))).Add(gtx.Ops)
					rect := clip.Rect{Min: image.Pt(0, 0), Max: image.Pt(screenWidth, screenHeight)}
					paint.FillShape(gtx.Ops, color.NRGBA{B: 0xFF, A: 0xFF}, rect.Op())

					return layout.Dimensions{Size: image.Pt(screenWidth*.2, screenHeight)}
				}),
			)
			// Update display.
			e.Frame(gtx.Ops)
		}
	}
}
func drawScene(ops *op.Ops, q event.Queue, tag any, w, h int) {

	for _, ev := range q.Events(tag) {
		if x, ok := ev.(pointer.Event); ok {
			switch x.Kind {
			case pointer.Drag:
				fmt.Printf("DRAG: %d\n", dragPt)

				if dragPt >= 0 {
					if spline.curve == nil || spline == nil {
						pts[dragPt], pts[dragPt+1] = float64(x.Position.X), float64(x.Position.Y)
					} else {
						spline.curve.SetControlPointVec2At(dragPt/2, ts.NewVec2(float64(x.Position.X), float64(x.Position.Y)))
					}
					goto cont
				}
			case pointer.Press:
				for k := range pts {
					if k%2 == 1 {
						continue
					}
					dragPt = -1
					if In((pts[k]), (pts[k+1]), x.Position.X, x.Position.Y, 14) {
						fmt.Printf("IN: %d\n", k)
						dragPt = k
						if spline.curve == nil || spline == nil {
							pts[k], pts[k+1] = float64(x.Position.X), float64(x.Position.Y)
						} else {
							spline.curve.SetControlPointVec2At(k/2, ts.NewVec2(float64(x.Position.X), float64(x.Position.Y)))
						}
						goto cont
					}
				}
				pts = append(pts, float64(x.Position.X), float64(x.Position.Y))
				fmt.Printf("press %0.0f %0.0f, %v\n", x.Position.X, x.Position.Y, pts)
				spline.NewSpline(&pts)

			case pointer.Release:
				println("RELEASED")
				dragPt = -1
			}
		}
	}

cont:

	if len(pts) != 0 {

		if spline != nil && spline.curve != nil {
			pts = spline.curve.GetControlPoints()
		}
		if len(pts) >= 4 {
			p := DrawLine(&pts, 2, ops)
			c := color.NRGBA{G: 0xFF, A: 0xFF}
			paint.FillShape(ops, c, clip.Stroke{Path: p.End(), Width: 3}.Op())
			DrawSpline(spline, 30, ops)
		}
		DrawPoints(&pts, ops, color.NRGBA{G: 125, R: 125, A: 255})
	}
	area := clip.Rect(image.Rect(0, 0, w, h)).Push(ops)
	pointer.InputOp{
		Tag:          tag,
		Kinds:        pointer.Press | pointer.Release | pointer.Drag,
		ScrollBounds: image.Rect(0, 0, w, h),
	}.Add(ops)
	area.Pop()
	// defer clip.Rect{Max: image.Pt(100, 400)}.Push(ops).Pop()
	// paint.ColorOp{Color: color.NRGBA{G: 0xFF, A: 0xFF}}.Add(ops)
	// paint.PaintOp{}.Add(ops)
}
