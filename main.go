package main

import (
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
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/colorpicker"
	"gioui.org/x/component"
	ts "github.com/tinyspline/go"
)

const (
	screenWidth  = 960
	screenHeight = 1000
)

type App struct{}

func main() {
	// Start CPU profiling
	f, err := os.Create("cpu.pprof")
	if err != nil {
		panic(err)
	}
	pprof.StartCPUProfile(f)
	go func() {
		window := app.NewWindow(app.Size(unit.Dp(screenWidth), unit.Dp(screenHeight)))

		if err := draw(window); err != nil {
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

var showSettings bool
var state *colorpicker.State

func draw(window *app.Window) error {

	spline = &Sp{degree: 3}

	var ops op.Ops
	events := make(chan event.Event)
	acks := make(chan struct{})
	th := material.NewTheme()
	// clearBtn is a clickable widget
	var clearBtn widget.Clickable
	var transformBtn widget.Clickable
	var modalBtn widget.Clickable
	settingModal := component.NewModal()

	go func() {
		for {
			ev := window.NextEvent()
			events <- ev
			<-acks
			if _, ok := ev.(system.DestroyEvent); ok {
				return
			}
		}
	}()
	a1 := 0
	type C = layout.Context
	type D = layout.Dimensions

	state = &colorpicker.State{}
	state.SetColor(color.NRGBA{R: 0xFF, A: 0xFF})

	for {
		select {
		case e := <-events:
			switch e := e.(type) {
			case system.DestroyEvent:
				acks <- struct{}{}
				return e.Err
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)

				in := layout.UniformInset(unit.Dp(8))

				layout.Stack{Alignment: layout.Center}.Layout(gtx,
					layout.Expanded(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(
							gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								w, h := screenWidth*.8, screenHeight
								drawScene(gtx.Ops, gtx.Queue, a1, int(w), h)
								return layout.Dimensions{Size: image.Pt(screenWidth*.8, screenHeight)}
							}),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(
										func(gtx C) D {
											return in.Layout(gtx,
												func(gtx C) D {
													text := "Clear Scene"
													if clearBtn.Clicked(gtx) {
														spline.curve = nil
														pts = nil
														dragPt = -1
														window.Invalidate()
													}
													btn := material.Button(th, &clearBtn, text)
													return btn.Layout(gtx)
												},
											)
										},
									),
									layout.Rigid(
										func(gtx C) D {
											return in.Layout(gtx,
												func(gtx C) D {
													text := "To bezier curves"
													if transformBtn.Clicked(gtx) {
														settingModal.VisibilityAnimation.State = component.Appearing
													}
													btn := material.Button(th, &transformBtn, text)
													return btn.Layout(gtx)
												},
											)
										},
									),
								)
							}),
						)
					}))

				modalInset := layout.UniformInset(unit.Dp(88))
				settingModal.Widget = func(gtx layout.Context, th *material.Theme, anim *component.VisibilityAnimation) layout.Dimensions {
					println("HIII")
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return modalInset.Layout(gtx,
									func(gtx C) D {
										text := "To bezier curves"
										if modalBtn.Clicked(gtx) {
											settingModal.VisibilityAnimation.State = component.Invisible
										}
										btn := material.Button(th, &modalBtn, text)
										return btn.Layout(gtx)
									},
								)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return modalInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return colorpicker.Picker(th, state, "Pick the color").Layout(gtx)
								})
							}),
						)
					}),
					)
				}
				settingModal.Layout(gtx, th)
				e.Frame(gtx.Ops)
			}
			acks <- struct{}{}
		}
	}
}
func drawScene(ops *op.Ops, q event.Queue, tag any, w, h int) {

	for _, ev := range q.Events(tag) {
		if x, ok := ev.(pointer.Event); ok {
			switch x.Kind {
			case pointer.Drag:
				var xd float32
				var yd float32
				if x.Position.X > float32(w) {
					xd = float32(w)
				} else if x.Position.X < 0 {
					xd = 0
				} else {
					xd = x.Position.X
				}
				if x.Position.Y > float32(h) {
					yd = float32(h)
				} else if x.Position.Y < 0 {
					yd = 0
				} else {
					yd = x.Position.Y
				}

				if dragPt >= 0 {
					if spline.curve == nil || spline == nil {
						pts[dragPt], pts[dragPt+1] = float64(xd), float64(yd)
					} else {
						spline.curve.SetControlPointVec2At(dragPt/2, ts.NewVec2(float64(xd), float64(yd)))
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
				spline.NewSpline(&pts)

			case pointer.Release:
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
		Tag:   tag,
		Kinds: pointer.Press | pointer.Release | pointer.Drag,
		// ScrollBounds: image.Rect(0, 0, w, h),
	}.Add(ops)
	area.Pop()
}
