// Copyright 2021 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package scbuild

import (
	"context"

	"github.com/cockroachdb/cockroach/pkg/sql/catalog/descpb"
	"github.com/cockroachdb/cockroach/pkg/sql/privilege"
	"github.com/cockroachdb/cockroach/pkg/sql/schemachanger/scpb"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
)

func (b *buildContext) dropDatabase(ctx context.Context, n *tree.DropDatabase) {
	// Check that the database exists.
	dbDesc, err := b.Descs.GetMutableDatabaseByName(ctx, b.EvalCtx.Txn, string(n.Name),
		tree.DatabaseLookupFlags{Required: !n.IfExists})
	if err != nil {
		panic(err)
	}
	if dbDesc == nil {
		// IfExists was specified and database was not found.
		return
	}

	if err := b.AuthAccessor.CheckPrivilege(ctx, dbDesc, privilege.DROP); err != nil {
		panic(err)
	}
	schemas, err := b.Descs.GetSchemasForDatabase(ctx, b.EvalCtx.Txn, dbDesc.GetID())
	if err != nil {
		panic(err)
	}

	dropIDs := make([]descpb.ID, 0, len(schemas))
	for schemaID := range schemas {
		dropIDs = append(dropIDs, schemaID)
		schemaDesc, err := b.Descs.GetImmutableSchemaByID(ctx, b.EvalCtx.Txn, schemaID, tree.SchemaLookupFlags{Required: true})
		//		if !n.DropBehavior FIXME: Error out here.
		if err != nil {
			panic(err)
		}
		nodeAdded, schemaDroppedIDs := b.dropSchemaDesc(ctx, schemaDesc, dbDesc, n.DropBehavior)
		// If no schema exists to depend on, then depend on dropped IDs
		if !nodeAdded {
			dropIDs = append(dropIDs, schemaDroppedIDs...)
		}
	}
	b.addNode(scpb.Target_DROP,
		&scpb.Database{
			DatabaseID:       dbDesc.ID,
			DependentObjects: dropIDs})
}
