package browser

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/input"
)

// MouseTrajectory represents a point in mouse movement
type MouseTrajectory struct {
	X, Y float64
	Time time.Duration
}

// GhostCursor implements realistic mouse movement similar to ghost-cursor
type GhostCursor struct {
	currentX, currentY float64
	random             *rand.Rand
}

// NewGhostCursor creates a new ghost cursor instance
func NewGhostCursor() *GhostCursor {
	return &GhostCursor{
		currentX: 0,
		currentY: 0,
		random:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateTrajectory generates a realistic mouse trajectory from current position to target
func (gc *GhostCursor) GenerateTrajectory(targetX, targetY float64) []MouseTrajectory {
	if gc.currentX == 0 && gc.currentY == 0 {
		// Initialize current position if not set
		gc.currentX = 100 + gc.random.Float64()*200
		gc.currentY = 100 + gc.random.Float64()*200
	}

	startX, startY := gc.currentX, gc.currentY
	deltaX := targetX - startX
	deltaY := targetY - startY
	distance := math.Sqrt(deltaX*deltaX + deltaY*deltaY)

	// Calculate number of steps based on distance
	steps := int(math.Max(10, distance/10))
	if steps > 100 {
		steps = 100 // Cap at 100 steps
	}

	trajectory := make([]MouseTrajectory, steps)
	
	// Generate Bezier curve points for realistic movement
	controlX1 := startX + deltaX*0.25 + (gc.random.Float64()-0.5)*distance*0.1
	controlY1 := startY + deltaY*0.25 + (gc.random.Float64()-0.5)*distance*0.1
	controlX2 := startX + deltaX*0.75 + (gc.random.Float64()-0.5)*distance*0.1
	controlY2 := startY + deltaY*0.75 + (gc.random.Float64()-0.5)*distance*0.1

	totalTime := time.Duration(200+distance/5) * time.Millisecond

	for i := 0; i < steps; i++ {
		t := float64(i) / float64(steps-1)
		
		// Cubic Bezier curve calculation
		x := math.Pow(1-t, 3)*startX + 3*math.Pow(1-t, 2)*t*controlX1 + 
			3*(1-t)*math.Pow(t, 2)*controlX2 + math.Pow(t, 3)*targetX
		y := math.Pow(1-t, 3)*startY + 3*math.Pow(1-t, 2)*t*controlY1 + 
			3*(1-t)*math.Pow(t, 2)*controlY2 + math.Pow(t, 3)*targetY

		// Add small random variations for more realistic movement
		x += (gc.random.Float64() - 0.5) * 2
		y += (gc.random.Float64() - 0.5) * 2

		// Calculate timing with easing
		timeRatio := gc.easeInOutQuad(t)
		stepTime := time.Duration(float64(totalTime) * timeRatio)

		trajectory[i] = MouseTrajectory{
			X:    x,
			Y:    y,
			Time: stepTime,
		}
	}

	// Update current position
	gc.currentX = targetX
	gc.currentY = targetY

	return trajectory
}

// easeInOutQuad provides smooth acceleration and deceleration
func (gc *GhostCursor) easeInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}

// RealClick performs a realistic click with human-like mouse movement
func (p *CDPPage) RealClick(x, y float64) error {
	cursor := NewGhostCursor()
	
	// Generate trajectory to target position
	trajectory := cursor.GenerateTrajectory(x, y)
	
	return chromedp.Run(p.ctx,
		// Move mouse along trajectory
		chromedp.ActionFunc(func(ctx context.Context) error {
			for i, point := range trajectory {
				// Move mouse to point
				if err := input.DispatchMouseEvent(input.MouseMoved, point.X, point.Y).Do(ctx); err != nil {
					return err
				}
				
				// Add realistic timing delays between movements
				if i < len(trajectory)-1 {
					delay := trajectory[i+1].Time - point.Time
					if delay > 0 {
						time.Sleep(delay)
					} else {
						time.Sleep(5 * time.Millisecond) // Minimum delay
					}
				}
			}
			return nil
		}),
		
		// Perform the actual click with realistic timing
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Mouse down
			if err := input.DispatchMouseEvent(input.MousePressed, x, y).
				WithButton(input.Left).
				WithClickCount(1).Do(ctx); err != nil {
				return err
			}
			
			// Realistic click duration (human clicks typically last 50-200ms)
			clickDuration := time.Duration(50+rand.Intn(150)) * time.Millisecond
			time.Sleep(clickDuration)
			
			// Mouse up
			return input.DispatchMouseEvent(input.MouseReleased, x, y).
				WithButton(input.Left).
				WithClickCount(1).Do(ctx)
		}),
	)
}

// RealHover performs a realistic hover with human-like mouse movement
func (p *CDPPage) RealHover(x, y float64) error {
	cursor := NewGhostCursor()
	trajectory := cursor.GenerateTrajectory(x, y)
	
	return chromedp.Run(p.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			for i, point := range trajectory {
				if err := input.DispatchMouseEvent(input.MouseMoved, point.X, point.Y).Do(ctx); err != nil {
					return err
				}
				
				if i < len(trajectory)-1 {
					delay := trajectory[i+1].Time - point.Time
					if delay > 0 {
						time.Sleep(delay)
					} else {
						time.Sleep(5 * time.Millisecond)
					}
				}
			}
			return nil
		}),
	)
}

// RealScroll performs realistic scrolling with human-like variations
func (p *CDPPage) RealScroll(deltaX, deltaY float64) error {
	// Add random variations to make scrolling more human-like
	variationX := deltaX + (rand.Float64()-0.5)*deltaX*0.1
	variationY := deltaY + (rand.Float64()-0.5)*deltaY*0.1
	
	return chromedp.Run(p.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return input.DispatchMouseEvent(input.MouseWheel, 0, 0).
				WithDeltaX(variationX).
				WithDeltaY(variationY).Do(ctx)
		}),
	)
}

// RealType performs realistic typing with human-like timing and variations
func (p *CDPPage) RealType(text string) error {
	return chromedp.Run(p.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			for _, char := range text {
				// Type character using proper input method
				if err := input.DispatchKeyEvent(input.KeyDown).
					WithText(string(char)).Do(ctx); err != nil {
					return err
				}
				if err := input.DispatchKeyEvent(input.KeyUp).
					WithText(string(char)).Do(ctx); err != nil {
					return err
				}
				
				// Add realistic typing delay (humans type at 200-500ms per character)
				delay := time.Duration(100+rand.Intn(150)) * time.Millisecond
				time.Sleep(delay)
			}
			return nil
		}),
	)
}