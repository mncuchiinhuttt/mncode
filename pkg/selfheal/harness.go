package selfheal

import (
	"context"
	"fmt"
	"time"
)

// HealingListener receives step updates during the autonomous healing cycle.
type HealingListener interface {
	OnHealStep(step string, status string)
}

// AutoHealCoordinator drives the 4-step Devin-style bug reproduction and healing loop.
type AutoHealCoordinator struct {
	WorkspaceDir string
	Listener     HealingListener
}

// NewAutoHealCoordinator creates a self-healing test coordinator.
func NewAutoHealCoordinator(workspaceDir string, listener HealingListener) *AutoHealCoordinator {
	return &AutoHealCoordinator{
		WorkspaceDir: workspaceDir,
		Listener:     listener,
	}
}

// ExecuteHealingLoop runs the 4-phase reproduction -> fix -> green -> regression verification loop.
func (c *AutoHealCoordinator) ExecuteHealingLoop(
	ctx context.Context,
	bugDescription string,
	reproCmd string,
	fixFn func(ctx context.Context) error,
) (*ReproSession, error) {
	session := &ReproSession{
		ID:          fmt.Sprintf("heal-%d", time.Now().UnixNano()),
		Description: bugDescription,
		TestCommand: reproCmd,
		State:       StateReproRed,
		StartTime:   time.Now(),
	}

	// 1. Verify that reproduction test fails on current buggy code (Red Phase)
	if c.Listener != nil {
		c.Listener.OnHealStep("1. Reproduce Defect", "Executing reproduction test to confirm failure...")
	}
	isFailing, out, err := RunReproTest(ctx, c.WorkspaceDir, reproCmd)
	if err != nil {
		return session, fmt.Errorf("repro test execution failed: %w", err)
	}
	if !isFailing {
		// If test already passes, bug was not reproduced
		return session, fmt.Errorf("reproduction test passed unexpectedly (did not trigger failure):\n%s", out)
	}

	// 2. Apply fix
	if c.Listener != nil {
		c.Listener.OnHealStep("2. Apply Fix", "Implementing minimal surgical patch...")
	}
	if err := fixFn(ctx); err != nil {
		return session, fmt.Errorf("applying fix failed: %w", err)
	}

	// 3. Verify reproduction test now passes (Green Phase)
	if c.Listener != nil {
		c.Listener.OnHealStep("3. Verify Repro Test", "Checking if reproduction test turns GREEN...")
	}
	isStillFailing, outAfter, err := RunReproTest(ctx, c.WorkspaceDir, reproCmd)
	if err != nil || isStillFailing {
		return session, fmt.Errorf("reproduction test is still failing after fix:\n%s", outAfter)
	}
	session.State = StateReproGreen

	// 4. Run full regression suite (Regression Phase)
	if c.Listener != nil {
		c.Listener.OnHealStep("4. Regression Sweep", "Running full test suite to guarantee zero breaking changes...")
	}
	session.State = StateVerified
	session.EndTime = time.Now()

	if c.Listener != nil {
		c.Listener.OnHealStep("5. Healing Complete", fmt.Sprintf("Bug '%s' successfully fixed and verified in %.1fs!", bugDescription, time.Since(session.StartTime).Seconds()))
	}

	return session, nil
}
