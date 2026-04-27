package reporter

import (
	"encoding/json"
	"io"

	"github.com/che1nov/gopulse/internal/domain"
)

type JSON struct{}

func NewJSON() JSON {
	return JSON{}
}

func (r JSON) PrintSnapshot(w io.Writer, snapshot domain.Snapshot) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(snapshot)
}

func (r JSON) PrintCheck(w io.Writer, result domain.CheckResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
