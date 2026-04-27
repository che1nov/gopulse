package usecases

import (
	"context"
	"io"

	"github.com/che1nov/gopulse/internal/domain"
)

type SnapshotReporter interface {
	PrintSnapshot(w io.Writer, snapshot domain.Snapshot) error
	PrintCheck(w io.Writer, result domain.CheckResult) error
}

type GenerateReport struct {
	checker CheckRegression
}

func NewGenerateReport(checker CheckRegression) GenerateReport {
	return GenerateReport{checker: checker}
}

func (uc GenerateReport) Execute(ctx context.Context, cfg Config) (domain.CheckResult, error) {
	return uc.checker.Execute(ctx, cfg)
}
