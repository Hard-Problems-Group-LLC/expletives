package expletives

import (
	"context"
	"errors"
	"testing"
)

func TestRootConstraintsDefaultMaximumAndAspectRatio(t *testing.T) {
	app := mustApp(t, Size{Width: 80, Height: 24})
	if got, want := app.Root().Bounds(), (Rect{Width: 80, Height: 24}); got != want {
		t.Fatalf("default root bounds = %+v, want %+v", got, want)
	}

	constraints := RootConstraints{
		Minimum:     Size{Width: 20, Height: 8},
		Maximum:     Size{Width: 40, Height: 18},
		AspectRatio: AspectRatio{Width: 16, Height: 9},
	}
	if err := app.SetRootConstraints(constraints); err != nil {
		t.Fatalf("SetRootConstraints() error = %v", err)
	}
	if got, want := app.RootConstraints(), constraints; got != want {
		t.Fatalf("RootConstraints() = %+v, want %+v", got, want)
	}
	if got, want := app.Root().Bounds(), (Rect{
		X: 24, Y: 3, Width: 32, Height: 18,
	}); got != want {
		t.Fatalf("constrained root bounds = %+v, want %+v", got, want)
	}
	if got := app.Root().MinimumSize(); got != constraints.Minimum {
		t.Fatalf("root minimum = %+v, want %+v", got, constraints.Minimum)
	}
}

func TestRootConstraintsBatchAtomicallyWithSurfaceResize(t *testing.T) {
	app := mustApp(t, Size{Width: 20, Height: 10})
	before := app.Snapshot().Sequence
	tx := app.NewTransaction()
	if err := tx.SetSize(Size{Width: 100, Height: 40}); err != nil {
		t.Fatalf("SetSize() error = %v", err)
	}
	if err := tx.SetRootConstraints(RootConstraints{
		Maximum: Size{Width: 60, Height: 20},
	}); err != nil {
		t.Fatalf("SetRootConstraints() error = %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if got, want := app.Root().Bounds(), (Rect{
		X: 20, Y: 10, Width: 60, Height: 20,
	}); got != want {
		t.Fatalf("root bounds = %+v, want %+v", got, want)
	}
	if got := app.Snapshot().Sequence; got != before+1 {
		t.Fatalf("sequence = %d, want one atomic publication after %d", got, before)
	}
}

func TestRootMinimumDoesNotEnlargePhysicalSurface(t *testing.T) {
	app, err := NewApp(AppOptions{
		Size: Size{Width: 10, Height: 4},
		RootConstraints: RootConstraints{
			Minimum: Size{Width: 20, Height: 8},
		},
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	if got, want := app.Root().Bounds(), (Rect{Width: 10, Height: 4}); got != want {
		t.Fatalf("root bounds = %+v, want available surface %+v", got, want)
	}
	if got, want := app.Root().MinimumSize(), (Size{Width: 20, Height: 8}); got != want {
		t.Fatalf("root minimum = %+v, want %+v", got, want)
	}
}

func TestRootConstraintsRejectContradictions(t *testing.T) {
	app := mustApp(t, Size{Width: 20, Height: 10})
	if err := app.SetRootConstraints(RootConstraints{
		Minimum: Size{Width: 10},
		Maximum: Size{Width: 9},
	}); !errors.Is(err, ErrInvalidGeometry) {
		t.Fatalf("minimum above maximum error = %v, want ErrInvalidGeometry", err)
	}
	if err := app.SetRootConstraints(RootConstraints{
		AspectRatio: AspectRatio{Width: 16},
	}); !errors.Is(err, ErrInvalidGeometry) {
		t.Fatalf("partial aspect ratio error = %v, want ErrInvalidGeometry", err)
	}
	if err := app.Root().SetMinimumSize(Size{Width: 1}); err == nil {
		t.Fatal("root SetMinimumSize() succeeded instead of directing caller to root constraints")
	}
}
