package deployment

import (
	"context"
	"fmt"
)

// rollbackLab removes staged/pointed Lab artifacts after a downstream
// failure. The caller already holds the publication failure that matters, so
// rollback errors are wrapped and returned for the caller to attach without
// masking the original failure.
func rollbackLab(ctx context.Context, lab LabPublisher, staged LabStaged) error {
	if lab == nil {
		return nil
	}
	if err := lab.Rollback(ctx, staged); err != nil {
		return fmt.Errorf("deployment: rollback lab: %w", err)
	}
	return nil
}

// rollbackPortfolio mirrors rollbackLab for the portfolio catalog entry.
func rollbackPortfolio(ctx context.Context, portfolio PortfolioCatalogPublisher, staged PortfolioStaged) error {
	if portfolio == nil {
		return nil
	}
	if err := portfolio.Rollback(ctx, staged); err != nil {
		return fmt.Errorf("deployment: rollback portfolio: %w", err)
	}
	return nil
}

// RollbackRequest names the staged versions to unwind. Nil sides are
// skipped, so callers roll back only the destinations that moved.
type RollbackRequest struct {
	Lab       *LabStaged
	Portfolio *PortfolioStaged
}

// Rollback removes staged or pointed artifacts for one publication from both
// destinations. It is the explicit entry point Task 4 wiring (or an
// operator) calls when publication must be unwound outside the Publish path
// itself. Both sides run even if the first fails; the first error is
// returned.
func Rollback(ctx context.Context, lab LabPublisher, portfolio PortfolioCatalogPublisher, req RollbackRequest) error {
	var first error
	if lab != nil && req.Lab != nil {
		if err := lab.Rollback(ctx, *req.Lab); err != nil {
			first = fmt.Errorf("deployment: rollback lab: %w", err)
		}
	}
	if portfolio != nil && req.Portfolio != nil {
		if err := portfolio.Rollback(ctx, *req.Portfolio); err != nil {
			if first == nil {
				first = fmt.Errorf("deployment: rollback portfolio: %w", err)
			}
		}
	}
	return first
}
