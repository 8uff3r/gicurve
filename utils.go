package main

import (
	"gioui.org/f32"
	"math"
)

const (
	rad90 = float32(90 * math.Pi / 180)
)

func offsetPoint(point f32.Point, distance, angle float32) f32.Point {
	x := point.X + distance*cos(angle)
	y := point.Y + distance*sin(angle)
	return f32.Point{X: x, Y: y}
}
func angle(p1, p2 f32.Point) float32 {
	return atan2(p2.Y-p1.Y, p2.X-p1.X)
}

func cos(v float32) float32 {
	return float32(math.Cos(float64(v)))
}

func sin(v float32) float32 {
	return float32(math.Sin(float64(v)))
}

func min(x, y float32) float32 {
	return float32(math.Min(float64(x), float64(y)))
}

func max(x, y float32) float32 {
	return float32(math.Max(float64(x), float64(y)))
}

func atan2(y, x float32) float32 {
	return float32(math.Atan2(float64(y), float64(x)))
}
func IntPow(n, m int) int {
	if m == 0 {
		return 1
	}
	result := n
	for i := 2; i <= m; i++ {
		result *= n
	}
	return result
}
func In(a, b float64, x, y float32, r int) bool {
	xf := float64(x)
	yf := float64(y)
	return math.Sqrt(math.Pow(xf-a, 2)+math.Pow(yf-b, 2))-float64(r)-2 <= 0
}
func RmElem[T any](slice []T, from, num int) []T {
	return append(slice[:from], slice[from+num:]...)
}
