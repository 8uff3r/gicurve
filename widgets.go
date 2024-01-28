package main

import (
	"image"
	"image/color"
	"time"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"gioui.org/x/colorpicker"
	"gioui.org/x/component"
)

type ColorPickerWithModal struct {
	modal *component.ModalLayer
	curr  color.NRGBA
}
type C = layout.Context
type D = layout.Dimensions

var cpick *colorpicker.PickerStyle

func NewColorPickerWithModal(initialColor color.NRGBA) *ColorPickerWithModal {
	res := &ColorPickerWithModal{
		curr:  initialColor,
		modal: component.NewModal(),
	}
	return res
}

func (c *ColorPickerWithModal) ToggleModal() {
	c.modal.ToggleVisibility(time.Now())
}

func (c *ColorPickerWithModal) SetColor(color color.NRGBA) {
	c.curr = color
}

var notFirstRun bool

func (c *ColorPickerWithModal) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {

	c.modal.Duration = time.Duration(time.Millisecond * 1)
	modalInset := layout.Inset{Top: unit.Dp(gtx.Constraints.Max.Y/2 - 100), Bottom: unit.Dp(gtx.Constraints.Max.Y/2 - 100), Left: 300, Right: 300}
	modalContentInset := layout.Inset{Top: modalInset.Top + 2, Bottom: modalInset.Bottom + 2, Left: 302, Right: 302}
	c.modal.Widget = func(gtx C, th *material.Theme, anim *component.VisibilityAnimation) D {
		return layout.Stack{Alignment: layout.Center}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				return modalInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					defer clip.RRect{Rect: image.Rect(gtx.Constraints.Min.X, gtx.Constraints.Min.Y, gtx.Constraints.Max.X, gtx.Constraints.Max.Y), SE: 5, SW: 5, NW: 5, NE: 5}.Push(gtx.Ops).Pop()
					paint.ColorOp{Color: color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}}.Add(gtx.Ops)
					paint.PaintOp{}.Add(gtx.Ops)
					return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X-gtx.Constraints.Min.X, gtx.Constraints.Max.Y-gtx.Constraints.Min.Y)}
				})
			}),
			layout.Expanded(
				func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Flexed(1, func(gtx C) D {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx C) D {
									return modalContentInset.Layout(gtx, func(gtx C) D {
										cpick := colorpicker.Picker(th, state, "Pick the color")
										if !notFirstRun {
											cpick.SetColor(c.curr)
										}
										notFirstRun = true
										c.curr = cpick.Color()
										return cpick.Layout(gtx)
									})
								}),
							)
						}),
					)
				},
			),
		)
	}
	return c.modal.Layout(gtx, th)
}
