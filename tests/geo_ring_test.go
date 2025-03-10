// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package tests

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/paulmach/orb"
	"github.com/stretchr/testify/assert"
)

func TestGeoRing(t *testing.T) {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"127.0.0.1:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
			Compression: &clickhouse.Compression{
				Method: clickhouse.CompressionLZ4,
			},
			Settings: clickhouse.Settings{
				"allow_experimental_geo_types": 1,
			},
		})
	)
	require.NoError(t, err)
	if err := CheckMinServerVersion(conn, 21, 12, 0); err != nil {
		t.Skip(err.Error())
		return
	}
	const ddl = `
		CREATE TABLE test_geo_ring (
			Col1 Ring
			, Col2 Array(Ring)
		) Engine Memory
		`
	defer func() {
		conn.Exec(ctx, "DROP TABLE test_geo_ring")
	}()
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_geo_ring")
	require.NoError(t, err)
	var (
		col1Data = orb.Ring{
			orb.Point{1, 2},
			orb.Point{1, 2},
		}
		col2Data = []orb.Ring{
			orb.Ring{
				orb.Point{1, 2},
				orb.Point{1, 2},
			},
			orb.Ring{
				orb.Point{1, 2},
				orb.Point{1, 2},
			},
		}
	)
	require.NoError(t, batch.Append(col1Data, col2Data))
	require.NoError(t, batch.Send())
	var (
		col1 orb.Ring
		col2 []orb.Ring
	)
	require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_geo_ring").Scan(&col1, &col2))
	assert.Equal(t, col1Data, col1)
	assert.Equal(t, col2Data, col2)
}

func TestGeoRingFlush(t *testing.T) {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"127.0.0.1:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
			Compression: &clickhouse.Compression{
				Method: clickhouse.CompressionLZ4,
			},
			Settings: clickhouse.Settings{
				"allow_experimental_geo_types": 1,
			},
		})
	)
	require.NoError(t, err)
	if err := CheckMinServerVersion(conn, 21, 12, 0); err != nil {
		t.Skip(err.Error())
		return
	}
	const ddl = `
		CREATE TABLE test_geo_ring_flush (
			  Col1 Ring
		) Engine Memory
		`
	defer func() {
		conn.Exec(ctx, "DROP TABLE test_geo_ring_flush")
	}()
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_geo_ring_flush")
	require.NoError(t, err)
	vals := [1000]orb.Ring{}
	for i := 0; i < 1000; i++ {
		vals[i] = orb.Ring{
			orb.Point{1, 2},
			orb.Point{1, 2},
		}
		require.NoError(t, batch.Append(vals[i]))
		require.NoError(t, batch.Flush())
	}
	require.NoError(t, batch.Send())
	rows, err := conn.Query(ctx, "SELECT * FROM test_geo_ring_flush")
	require.NoError(t, err)
	i := 0
	for rows.Next() {
		var col1 orb.Ring
		require.NoError(t, rows.Scan(&col1))
		require.Equal(t, vals[i], col1)
		i += 1
	}
	require.Equal(t, 1000, i)
}
