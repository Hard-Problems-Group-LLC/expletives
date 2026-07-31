package expletives

import "fmt"

func validateRootConstraints(constraints RootConstraints) error {
	if !constraints.Minimum.valid() || !constraints.Maximum.valid() {
		return fmt.Errorf("%w: negative root constraint", ErrInvalidGeometry)
	}
	if _, err := frameCellCount(constraints.Minimum); err != nil {
		return fmt.Errorf("minimum root size: %w", err)
	}
	if constraints.Maximum.Width != 0 &&
		constraints.Minimum.Width > constraints.Maximum.Width {
		return fmt.Errorf("%w: root minimum width exceeds maximum", ErrInvalidGeometry)
	}
	if constraints.Maximum.Height != 0 &&
		constraints.Minimum.Height > constraints.Maximum.Height {
		return fmt.Errorf("%w: root minimum height exceeds maximum", ErrInvalidGeometry)
	}
	ratio := constraints.AspectRatio
	if (ratio.Width == 0) != (ratio.Height == 0) ||
		ratio.Width < 0 || ratio.Height < 0 ||
		ratio.Width > MaxFrameCells || ratio.Height > MaxFrameCells {
		return fmt.Errorf("%w: aspect ratio requires two positive bounded terms", ErrInvalidGeometry)
	}
	if constraints.Maximum.Width > MaxFrameCells ||
		constraints.Maximum.Height > MaxFrameCells {
		return fmt.Errorf("%w: root maximum exceeds allocation dimension", ErrInvalidGeometry)
	}
	return nil
}

func constrainedRootBounds(surface Size, constraints RootConstraints) Rect {
	width, height := surface.Width, surface.Height
	if maximum := constraints.Maximum.Width; maximum != 0 {
		width = min(width, maximum)
	}
	if maximum := constraints.Maximum.Height; maximum != 0 {
		height = min(height, maximum)
	}

	ratio := constraints.AspectRatio
	if width > 0 && height > 0 && ratio.Width > 0 {
		ratioHeight := roundedRatio(width, ratio.Height, ratio.Width)
		if ratioHeight <= height {
			height = max(1, ratioHeight)
		} else {
			width = max(1, roundedRatio(height, ratio.Width, ratio.Height))
		}
	}

	return Rect{
		X:      (surface.Width - width) / 2,
		Y:      (surface.Height - height) / 2,
		Width:  width,
		Height: height,
	}
}

func roundedRatio(value, numerator, denominator int) int {
	product := int64(value) * int64(numerator)
	return int((product + int64(denominator)/2) / int64(denominator))
}
