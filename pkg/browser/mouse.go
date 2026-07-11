package browser

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
)

// MouseTrajectory represents a point in mouse movement
type MouseTrajectory struct {
	X, Y float64
	Time time.Duration
}

// GhostCursor implements realistic mouse movement similar to ghost-cursor.
// A cursor is persistent per page so consecutive moves continue from the last
// position instead of teleporting from a fresh random origin.
type GhostCursor struct {
	mu                 sync.Mutex
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
// 生成贝塞尔曲线鼠标轨迹（默认正常拟人速度）
func (gc *GhostCursor) GenerateTrajectory(targetX, targetY float64) []MouseTrajectory {
	return gc.GenerateTrajectoryWithSpeed(targetX, targetY, 2.0) // 2.0 = 正常拟人（1.0 快速时序过于机械，易被指纹识别）
}

// GenerateTrajectoryWithSpeed generates trajectory with speed multiplier.
// speedMultiplier: 1.0 = 快速, 2.0 = 正常拟人, 3.0 = 慢速。
// 导出的调速接口，供需要自定义节奏的调用方使用。
func (gc *GhostCursor) GenerateTrajectoryWithSpeed(targetX, targetY float64, speedMultiplier float64) []MouseTrajectory {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if gc.currentX == 0 && gc.currentY == 0 {
		// Initialize current position if not set
		gc.currentX = 100 + gc.random.Float64()*200
		gc.currentY = 100 + gc.random.Float64()*200
	}

	startX, startY := gc.currentX, gc.currentY
	deltaX := targetX - startX
	deltaY := targetY - startY
	distance := math.Sqrt(deltaX*deltaX + deltaY*deltaY)

	// 优化：减少步数 (5-25步，之前是 10-100步)
	steps := int(math.Max(5, distance/40))
	if steps > 25 {
		steps = 25
	}

	trajectory := make([]MouseTrajectory, steps)

	// Generate Bezier curve points for realistic movement
	controlX1 := startX + deltaX*0.25 + (gc.random.Float64()-0.5)*distance*0.1
	controlY1 := startY + deltaY*0.25 + (gc.random.Float64()-0.5)*distance*0.1
	controlX2 := startX + deltaX*0.75 + (gc.random.Float64()-0.5)*distance*0.1
	controlY2 := startY + deltaY*0.75 + (gc.random.Float64()-0.5)*distance*0.1

	// 优化：减少总时间 (50ms 基础 + 距离/20)，之前是 200ms + 距离/5
	baseTime := 50.0 * speedMultiplier
	totalTime := time.Duration(baseTime+distance/20*speedMultiplier) * time.Millisecond

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
	// Generate trajectory to target position (persistent cursor avoids teleporting)
	trajectory := p.getCursor().GenerateTrajectory(x, y)

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
						time.Sleep(2 * time.Millisecond) // 最小延迟 2ms
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

			// 优化：点击持续时间 10-60ms（之前 50-200ms）
			clickDuration := time.Duration(10+rand.Intn(50)) * time.Millisecond
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
	trajectory := p.getCursor().GenerateTrajectory(x, y)

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
						time.Sleep(2 * time.Millisecond) // 最小延迟 2ms
					}
				}
			}
			return nil
		}),
	)
}

// RealScroll performs realistic scrolling with human-like variations.
// Real wheels emit many small ticks; a single atomic delta is a bot signature,
// so the delta is split into ~100px ticks with jitter and inter-tick delays.
func (p *CDPPage) RealScroll(deltaX, deltaY float64) error {
	steps := wheelSteps(deltaX, deltaY)
	return chromedp.Run(p.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			for i, s := range steps {
				if err := input.DispatchMouseEvent(input.MouseWheel, 0, 0).
					WithDeltaX(s[0]).WithDeltaY(s[1]).Do(ctx); err != nil {
					return err
				}
				if i < len(steps)-1 {
					time.Sleep(time.Duration(15+rand.Intn(25)) * time.Millisecond)
				}
			}
			return nil
		}),
	)
}

// getCursor returns the page's persistent cursor, lazily creating it.
func (p *CDPPage) getCursor() *GhostCursor {
	if p.cursor == nil {
		p.cursor = NewGhostCursor()
	}
	return p.cursor
}

// wheelSteps splits a scroll delta into human-like wheel ticks (~100px each with
// ±15% jitter). Rounding drift is folded into the final tick so the total delta
// is exact. Always returns at least one tick.
func wheelSteps(deltaX, deltaY float64) [][2]float64 {
	dist := math.Max(math.Abs(deltaX), math.Abs(deltaY))
	n := max(int(math.Ceil(dist/100.0)), 1)
	steps := make([][2]float64, n)
	var accX, accY float64
	for i := range n {
		sx := deltaX / float64(n)
		sy := deltaY / float64(n)
		sx += (rand.Float64() - 0.5) * sx * 0.3
		sy += (rand.Float64() - 0.5) * sy * 0.3
		steps[i] = [2]float64{sx, sy}
		accX += sx
		accY += sy
	}
	steps[n-1][0] += deltaX - accX
	steps[n-1][1] += deltaY - accY
	return steps
}

// RealSendKeys performs realistic typing with human-like timing and variations
func (p *CDPPage) RealSendKeys(text string) error {
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

				// 优化：打字延迟 20-70ms（之前 100-250ms）
				delay := time.Duration(20+rand.Intn(50)) * time.Millisecond
				time.Sleep(delay)
			}
			return nil
		}),
	)
}
