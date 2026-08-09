package importing

import (
	"context"
	"time"
)

type SourceNode struct {
	SourceID int64
	Lat      float64
	Lon      float64
	Tags     map[string]string
}

type SourceWay struct {
	SourceID int64
	NodeIDs  []int64
	Tags     map[string]string
}

type SourceRelationMember struct {
	Kind     string
	SourceID int64
	Role     string
}

type SourceRelation struct {
	SourceID int64
	Members  []SourceRelationMember
	Tags     map[string]string
}

type SourceMetadata struct {
	Timestamp *time.Time
}

type SourceVisitor interface {
	VisitNode(SourceNode) error
	VisitWay(SourceWay) error
	VisitRelation(SourceRelation) error
}

type SourceScanner interface {
	Scan(context.Context, string, SourceVisitor) (SourceMetadata, error)
}
