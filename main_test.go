package main

import (
	"image/color"
	"testing"
)

func Test同じ色ならコンストラクト比は1になる(t *testing.T) {
	type TestCase struct {
		name  string
		color color.Color
	}
	colors := []TestCase{
		{name: "黒", color: color.Black},
		{name: "白", color: color.White},
		{name: "#ff0000", color: color.RGBA{R: 255, G: 0, B: 0, A: 1}},
	}

	for _, c := range colors {
		t.Run(c.name, func(t *testing.T) {
			if calcContrastRatio(c.color, c.color) != 1 {
				t.Errorf("fail")
			}
		})
	}
}

func Test白と黒ならコンストラクト比は21になる(t *testing.T) {
	if calcContrastRatio(color.Black, color.White) != 21 {
		t.Errorf("fail")
	}
}
