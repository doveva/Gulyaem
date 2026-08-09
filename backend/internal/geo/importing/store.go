package importing

import (
	"context"
	"errors"
	"time"

	"github.com/doveva/Gulyaem/backend/internal/geo/domain"
)

var (
	ErrCityNotFound     = errors.New("city not found")
	ErrImportInProgress = errors.New("another geo import is already running for this city")
)

type VersionStore interface {
	BeginImport(context.Context, domain.BeginImport) (domain.BeginImportResult, error)
	CompleteImport(context.Context, string, *time.Time, domain.ImportReport) (domain.GeoDataVersion, error)
	FailImport(context.Context, string, domain.ImportReport, error) error
}
