package districting

import (
	"context"
	"errors"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
)

var (
	ErrCityNotFound     = errors.New("city not found")
	ErrImportInProgress = errors.New("another district import is already running for this city")
)

type VersionStore interface {
	BeginImport(context.Context, domain.BeginDistrictImport) (domain.BeginDistrictImportResult, error)
	CompleteImport(context.Context, string, *time.Time, domain.DistrictImportReport, []domain.DistrictDraft) (domain.DistrictDataVersion, error)
	FailImport(context.Context, string, domain.DistrictImportReport, error) error
}
