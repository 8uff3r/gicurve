package main

import (
	"math"

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
	"strconv"

	"image"
	"image/color"
	"log"
	"os"
	"runtime/pprof"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	ts "github.com/tinyspline/go"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

const (
	screenWidth  = 970
	screenHeight = 720
)

const (
	Normal int = iota
	Rm
)

var State int

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

var interpolPts []float64
var isInterpolated bool
var interpolSpline *Sp

var runTime time.Time
var maxRunTime time.Duration
var uProgram *vm.Program
var maxVal any
var errorModal *component.ModalLayer
var eqModeSwitchIsPeriodic widget.Bool

var showSettings bool
var state *colorpicker.State

var sceneMode int

var xIcon *widget.Icon

func draw(window *app.Window) error {
	//TODO: Set this in input
	maxRunTime = time.Second * 4

	spline = &Sp{degree: 3}
	interpolSpline = &Sp{degree: 3}

	var ops op.Ops
	events := make(chan event.Event)
	acks := make(chan struct{})
	th := material.NewTheme()
	// clearBtn is a clickable widget
	var clearBtn widget.Clickable
	var transformBtn widget.Clickable
	var colorBtn widget.Clickable
	var deletePointBtn widget.Clickable
	var modalBtn widget.Clickable
	var swInterpolBtn widget.Clickable
	var swDrawBtn widget.Clickable
	var eqEditor widget.Editor
	var eqPEditor widget.Editor
	var durEditor widget.Editor
	eqEditor.Filter = "t1234567890+-*^"
	durEditor.Filter = "1234567890"

	var doInterpolBtn widget.Clickable
	var runMoverBtn widget.Clickable

	settingModal := component.NewModal()
	colorModal := NewColorPickerWithModal(th.ContrastBg)
	errorModal = component.NewModal()

	ic, _ := widget.NewIcon(icons.NavigationClose)
	xIcon = ic

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
					layout.Expanded(func(gtx C) D {
						return layout.Flex{Axis: layout.Horizontal}.Layout(
							gtx,
							layout.Rigid(func(gtx C) D {
								w, h := screenWidth*.75, screenHeight
								if sceneMode == 0 {

									DrawScene(gtx.Ops, gtx.Queue, a1, int(w), h)
								} else {
									InterpolScene(gtx.Ops, gtx.Queue, a1, int(w), h)
								}
								return D{Size: image.Pt(int(w), h)}
							}),
							layout.Flexed(1, func(gtx C) D {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return in.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return layout.Flex{Axis: layout.Horizontal}.Layout(
												gtx,
												layout.Flexed(0.5, func(gtx layout.Context) layout.Dimensions {
													if swDrawBtn.Clicked(gtx) {
														sceneMode = 0
													}
													btn := material.Button(th, &swDrawBtn, "Draw")
													btn.CornerRadius = 0
													if sceneMode != 0 {
														btn.Background = color.NRGBA{B: 250, A: 210}
													} else {
														btn.Background = color.NRGBA{B: 250, A: btn.Background.A}
													}
													return btn.Layout(gtx)
												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return layout.Spacer{Width: unit.Dp(4)}.Layout(gtx)
												}),

												layout.Flexed(0.5, func(gtx layout.Context) layout.Dimensions {
													if swInterpolBtn.Clicked(gtx) {
														sceneMode = 1
													}
													btn := material.Button(th, &swInterpolBtn, "Interpolate")
													btn.CornerRadius = 0
													if sceneMode != 1 {
														btn.Background = color.NRGBA{B: 250, A: 210}
													} else {
														btn.Background = color.NRGBA{B: 250, A: btn.Background.A}
													}
													return btn.Layout(gtx)
												}),
											)
										})
									}),
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
														spline.curve = spline.curve.ToBeziers()
													}
													btn := material.Button(th, &transformBtn, text)
													return btn.Layout(gtx)
												},
											)
										},
									),
									layout.Rigid(
										func(gtx C) D {
											return in.Layout(gtx,
												func(gtx C) D {
													text := "Change Colors"
													if colorBtn.Clicked(gtx) {
														colorModal.modal.VisibilityAnimation.ToggleVisibility(time.Now())
													}
													btn := material.Button(th, &colorBtn, text)
													return btn.Layout(gtx)
												},
											)
										},
									),
									layout.Rigid(
										func(gtx C) D {
											return in.Layout(gtx,
												func(gtx C) D {
													text := "Delete a point"
													if deletePointBtn.Clicked(gtx) {
														if State == Rm {
															State = Normal
														} else {
															State = Rm
														}
													}
													if len(pts) == 0 {
														State = Normal
														gtx = gtx.Disabled()
													}
													btn := material.Button(th, &deletePointBtn, text)
													return btn.Layout(gtx)
												},
											)
										},
									),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										if sceneMode == 1 {
											return layout.Dimensions{}
										}
										return in.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													dv := component.Divider(th)
													return dv.Layout(gtx)

												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													lbl := material.Label(th, unit.Sp(16), "Input move Equation")
													return lbl.Layout(gtx)

												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return layout.Spacer{Height: unit.Dp(9)}.Layout(gtx)
												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
														layout.Rigid(func(gtx layout.Context) layout.Dimensions {
															lbl := material.Label(th, unit.Sp(15), "Periodic")
															return lbl.Layout(gtx)
														}),
														layout.Rigid(func(gtx layout.Context) layout.Dimensions {
															return layout.Spacer{Width: unit.Dp(5)}.Layout(gtx)
														}),
														layout.Rigid(func(gtx layout.Context) layout.Dimensions {
															sw := material.Switch(th, &eqModeSwitchIsPeriodic, "Periodic")
															return sw.Layout(gtx)
														}),
													)
												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return layout.Spacer{Height: unit.Dp(9)}.Layout(gtx)
												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													if eqModeSwitchIsPeriodic.Value {
														return layout.Dimensions{}
													}
													return widget.Border{Color: color.NRGBA{A: 255, B: 180}, CornerRadius: unit.Dp(5), Width: unit.Dp(1)}.Layout(gtx,
														func(gtx layout.Context) layout.Dimensions {
															return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
																edtr := material.Editor(th, &eqEditor, "u(t)")
																edtr.TextSize = unit.Sp(18)

																return edtr.Layout(gtx)
															})
														})
												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													if !eqModeSwitchIsPeriodic.Value {
														return layout.Dimensions{}
													}
													return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
														layout.Rigid(func(gtx layout.Context) layout.Dimensions {
															lbl := material.Label(th, unit.Sp(18), "Sin(")
															return lbl.Layout(gtx)
														}),
														layout.Rigid(
															func(gtx layout.Context) layout.Dimensions {

																return widget.Border{Color: color.NRGBA{A: 255, B: 180}, CornerRadius: unit.Dp(5), Width: unit.Dp(1)}.Layout(gtx,
																	func(gtx layout.Context) layout.Dimensions {
																		return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
																			edtr := material.Editor(th, &eqPEditor, "u(t) = sin(f(t))")
																			edtr.TextSize = unit.Sp(18)

																			return edtr.Layout(gtx)
																		})
																	})
															}),
														layout.Rigid(func(gtx layout.Context) layout.Dimensions {
															lbl := material.Label(th, unit.Sp(18), ")")
															return lbl.Layout(gtx)
														}),
													)
												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return layout.Spacer{Height: unit.Dp(10)}.Layout(gtx)
												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													if runMoverBtn.Clicked(gtx) {
														runTime = time.Now()
														parsedMaxDuration, err := strconv.Atoi(durEditor.Text())
														if eqModeSwitchIsPeriodic.Value {
															uProgram, _ = expr.Compile(eqPEditor.Text())
														} else {

															uProgram, _ = expr.Compile(eqEditor.Text())
														}
														if err != nil {
															errorModal.VisibilityAnimation.State = component.Visible
														}
														maxRunTime = time.Second * time.Duration(parsedMaxDuration)
														env := map[string]float64{
															"t": maxRunTime.Seconds(),
														}
														maxVal, _ = expr.Run(uProgram, env)
													}
													if spline.curve == nil || (eqModeSwitchIsPeriodic.Value && len(eqPEditor.Text()) == 0) || (!eqModeSwitchIsPeriodic.Value && len(eqEditor.Text()) == 0) || len(durEditor.Text()) == 0 {
														gtx = gtx.Disabled()
													}
													btn := material.Button(th, &runMoverBtn, "Run")
													return btn.Layout(gtx)
												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return layout.Spacer{Height: unit.Dp(10)}.Layout(gtx)
												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													lbl := material.Label(th, unit.Sp(16), "Duration in seconds")
													return lbl.Layout(gtx)
												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return layout.Spacer{Height: unit.Dp(10)}.Layout(gtx)
												}),
												layout.Rigid(func(gtx layout.Context) layout.Dimensions {
													return widget.Border{Color: color.NRGBA{A: 255, B: 180}, CornerRadius: unit.Dp(5), Width: unit.Dp(1)}.Layout(gtx,
														func(gtx layout.Context) layout.Dimensions {
															return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
																edtr := material.Editor(th, &durEditor, "...seconds")
																edtr.TextSize = unit.Sp(18)

																return edtr.Layout(gtx)
															})
														})
												}),
											)
										})
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return in.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											if sceneMode == 0 {
												return layout.Dimensions{}
											}
											if doInterpolBtn.Clicked(gtx) {
												isInterpolated = true
											}
											btn := material.Button(th, &doInterpolBtn, "Interpolate Points")
											return btn.Layout(gtx)
										})
									}),
								)
							}),
						)
					}))
				colorModal.Layout(gtx, th)
				modalInset := layout.UniformInset(unit.Dp(88))
				modalContentInset := layout.UniformInset(unit.Dp(95))
				settingModal.Duration = time.Duration(time.Millisecond * 1)
				settingModal.Clickable = widget.Clickable{}
				settingModal.Widget = func(gtx C, th *material.Theme, anim *component.VisibilityAnimation) D {
					return layout.Stack{Alignment: layout.Center}.Layout(gtx,
						layout.Stacked(func(gtx layout.Context) layout.Dimensions {
							return modalInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								defer clip.RRect{Rect: image.Rect(gtx.Constraints.Min.X, gtx.Constraints.Min.Y, gtx.Constraints.Max.X, gtx.Constraints.Max.Y), SE: 5, SW: 5, NW: 5, NE: 5}.Push(gtx.Ops).Pop()
								paint.ColorOp{Color: color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}}.Add(gtx.Ops)
								paint.PaintOp{}.Add(gtx.Ops)
								return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X-gtx.Constraints.Min.X, gtx.Constraints.Max.Y-gtx.Constraints.Min.Y)}
							})
						}),
						layout.Stacked(func(gtx layout.Context) layout.Dimensions {
							btn := material.IconButton(th, &modalBtn, xIcon, "X")
							if modalBtn.Clicked(gtx) {
								settingModal.VisibilityAnimation.ToggleVisibility(time.Now())
							}
							btn.Inset = layout.UniformInset(3)
							x := +gtx.Constraints.Max.X/2 - int(modalInset.Left)
							y := -gtx.Constraints.Max.Y/2 + int(modalInset.Top)
							defer op.Offset(image.Pt(x, y)).Push(&ops).Pop()
							return btn.Layout(gtx)
						}),

						layout.Expanded(
							func(gtx C) D {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Flexed(1, func(gtx C) D {
										return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
											layout.Rigid(func(gtx C) D {
												return modalContentInset.Layout(gtx, func(gtx C) D {
													return colorpicker.Picker(th, state, "Pick the color").Layout(gtx)
												})
											}),
										)
									}),
								)
							},
						),
					)
				}
				settingModal.Layout(gtx, th)
				errorModal.Duration = time.Duration(time.Millisecond * 1)
				errorModal.Widget = func(gtx layout.Context, th *material.Theme, anim *component.VisibilityAnimation) layout.Dimensions {
					return modalInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(th, unit.Sp(18), "Expression is invalid")
							lbl.Color = color.NRGBA{R: 0xFF, A: 0xFF}
							return lbl.Layout(gtx)
						})
					})
				}
				errorModal.Layout(gtx, th)
				e.Frame(gtx.Ops)
			}
			acks <- struct{}{}
		}
	}
}
func DrawScene(ops *op.Ops, q event.Queue, tag any, w, h int) {
	DrawMover(ops)

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
					FillScene(ops, tag, w, h, &pts)
					return
				}
			case pointer.Press:
				dragPt = -1
				if State == Rm {
					var ctrlPts []float64
					if spline.curve != nil {
						ctrlPts = spline.curve.GetControlPoints()
					} else {
						ctrlPts = pts
					}

					for k := range ctrlPts {
						if k%2 == 1 {
							continue
						}
						if In((ctrlPts[k]), (ctrlPts[k+1]), x.Position.X, x.Position.Y, 14) {
							ctrlPts = RmElem(ctrlPts, k, 2)
							pts = ctrlPts
							err := spline.NewSpline(&ctrlPts)
							if err != nil {
								spline.curve = nil
							}
							break
						}
					}
					FillScene(ops, tag, w, h, &pts)
					return
				}
				for k := range pts {
					if k%2 == 1 {
						continue
					}
					if In((pts[k]), (pts[k+1]), x.Position.X, x.Position.Y, 14) {
						dragPt = k
						if spline.curve == nil || spline == nil {
							pts[k], pts[k+1] = float64(x.Position.X), float64(x.Position.Y)
						} else {
							spline.curve.SetControlPointVec2At(k/2, ts.NewVec2(float64(x.Position.X), float64(x.Position.Y)))
						}
						FillScene(ops, tag, w, h, &pts)
						return
					}
				}
				pts = append(pts, float64(x.Position.X), float64(x.Position.Y))
				spline.NewSpline(&pts)

			case pointer.Release:
				dragPt = -1
			}
		}
	}
	FillScene(ops, tag, w, h, &pts)
}

func DrawMover(ops *op.Ops) {
	if runTime.IsZero() {
		return
	}
	if time.Now().After(runTime.Add(maxRunTime)) {
		runTime = time.Time{}
		op.InvalidateOp{At: time.Time{}}.Add(ops)
		return
	}
	if spline.curve == nil {
		return
	}
	lapsed := time.Now().Sub(runTime)
	env := map[string]float64{
		"t": lapsed.Seconds(),
	}
	out, err := expr.Run(uProgram, env)
	if out == nil || err != nil {
		runTime = time.Time{}
		errorModal.VisibilityAnimation.ToggleVisibility(time.Now())
		return
	}
	switch out.(type) {
	case int:
		runTime = time.Time{}
		errorModal.VisibilityAnimation.State = component.Visible
		return

	}
	var u float64
	if eqModeSwitchIsPeriodic.Value {
		u = math.Abs(math.Sin(out.(float64)))
	} else {
		u = out.(float64) / maxVal.(float64)
	}
	net := spline.curve.Eval(u)
	pts := net.GetResult()
	op.InvalidateOp{At: time.Time{}}.Add(ops)
	DrawPoints(&pts, ops, color.NRGBA{R: 255, G: 140, A: 255}, 16)
}

func FillScene(ops *op.Ops, tag any, w, h int, pts *[]float64) {

	if len(*pts) != 0 {
		if spline != nil && spline.curve != nil {
			*pts = spline.curve.GetControlPoints()
		}
		if len(*pts) >= 4 {
			p := DrawLine(pts, 2, ops)
			c := color.NRGBA{G: 0xFF, A: 0xFF}
			paint.FillShape(ops, c, clip.Stroke{Path: p.End(), Width: 3}.Op())
			DrawSpline(spline, 30, ops)
		}
		DrawPoints(pts, ops, color.NRGBA{G: 125, R: 125, A: 255}, 8)
	}
	area := clip.Rect(image.Rect(0, 0, w, h)).Push(ops)
	pointer.InputOp{
		Tag:   tag,
		Kinds: pointer.Press | pointer.Release | pointer.Drag,
		// ScrollBounds: image.Rect(0, 0, w, h),
	}.Add(ops)
	area.Pop()
}

func InterpolScene(ops *op.Ops, q event.Queue, tag any, w, h int) {

	for _, ev := range q.Events(tag) {
		if x, ok := ev.(pointer.Event); ok {
			switch x.Kind {
			case pointer.Press:
				interpolPts = append(interpolPts, float64(x.Position.X), float64(x.Position.Y))
			}
		}
	}

	DrawPoints(&interpolPts, ops, color.NRGBA{G: 125, R: 125, A: 255}, 8)
	if isInterpolated {
		interpolSpline.curve = ts.BSplineInterpolateCatmullRom(interpolPts, 2)
		DrawSpline(interpolSpline, 30, ops)
		return
	}
	area := clip.Rect(image.Rect(0, 0, w, h)).Push(ops)
	pointer.InputOp{
		Tag:   tag,
		Kinds: pointer.Press | pointer.Release | pointer.Drag,
		// ScrollBounds: image.Rect(0, 0, w, h),
	}.Add(ops)
	area.Pop()
}
