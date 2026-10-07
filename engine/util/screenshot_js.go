//go:build js

package util

type AutoScreenshot struct{}

func NewAutoScreenshot() *AutoScreenshot { return nil }

func (s *AutoScreenshot) Update(elapsed float64, fbWidth, fbHeight int) {}
