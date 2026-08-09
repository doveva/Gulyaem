package pbf

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/doveva/Gulyaem/backend/internal/geo/importing"
	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
)

type Scanner struct {
	parallelism int
}

func NewScanner() *Scanner {
	return &Scanner{parallelism: runtime.GOMAXPROCS(0)}
}

func (source *Scanner) Scan(ctx context.Context, path string, visitor importing.SourceVisitor) (importing.SourceMetadata, error) {
	file, err := os.Open(path)
	if err != nil {
		return importing.SourceMetadata{}, err
	}
	defer file.Close()

	scanner := osmpbf.New(ctx, file, source.parallelism)
	defer scanner.Close()
	header, err := scanner.Header()
	if err != nil {
		return importing.SourceMetadata{}, fmt.Errorf("read PBF header: %w", err)
	}
	metadata := importing.SourceMetadata{}
	if !header.ReplicationTimestamp.IsZero() {
		timestamp := header.ReplicationTimestamp
		metadata.Timestamp = &timestamp
	}

	for scanner.Scan() {
		if err := visitObject(visitor, scanner.Object()); err != nil {
			return metadata, err
		}
	}
	if err := scanner.Err(); err != nil {
		return metadata, err
	}
	return metadata, nil
}

func visitObject(visitor importing.SourceVisitor, object osm.Object) error {
	switch value := object.(type) {
	case *osm.Node:
		return visitor.VisitNode(importing.SourceNode{
			SourceID: int64(value.ID),
			Lat:      value.Lat,
			Lon:      value.Lon,
			Tags:     value.Tags.Map(),
		})
	case *osm.Way:
		nodeIDs := make([]int64, len(value.Nodes))
		for index, node := range value.Nodes {
			nodeIDs[index] = int64(node.ID)
		}
		return visitor.VisitWay(importing.SourceWay{
			SourceID: int64(value.ID),
			NodeIDs:  nodeIDs,
			Tags:     value.Tags.Map(),
		})
	case *osm.Relation:
		members := make([]importing.SourceRelationMember, len(value.Members))
		for index, member := range value.Members {
			members[index] = importing.SourceRelationMember{
				Kind:     string(member.Type),
				SourceID: member.Ref,
				Role:     member.Role,
			}
		}
		return visitor.VisitRelation(importing.SourceRelation{
			SourceID: int64(value.ID),
			Members:  members,
			Tags:     value.Tags.Map(),
		})
	default:
		return fmt.Errorf("unsupported OSM object type %T", object)
	}
}
