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
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
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
			//Debug: true,
		})
	)
	require.NoError(t, err)
	if err := CheckMinServerVersion(conn, 21, 9, 0); err != nil {
		t.Skip(err.Error())
		return
	}
	const ddl = `
		CREATE TABLE test_map (
			  Col1 Map(String, UInt64)
			, Col2 Map(String, UInt64)
			, Col3 Map(String, UInt64)
			, Col4 Array(Map(String, String))
			, Col5 Map(LowCardinality(String), LowCardinality(UInt64))
			, Col6 Map(String, Map(String,UInt64))
		) Engine Memory
		`
	defer func() {
		conn.Exec(ctx, "DROP TABLE test_map")
	}()
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_map")
	require.NoError(t, err)
	var (
		col1Data = map[string]uint64{
			"key_col_1_1": 1,
			"key_col_1_2": 2,
		}
		col2Data = map[string]uint64{
			"key_col_2_1": 10,
			"key_col_2_2": 20,
		}
		col3Data = map[string]uint64{}
		col4Data = []map[string]string{
			map[string]string{"A": "B"},
			map[string]string{"C": "D"},
		}
		col5Data = map[string]uint64{
			"key_col_5_1": 100,
			"key_col_5_2": 200,
		}
		col6Data = map[string]map[string]uint64{
			"key_col_6_1": {
				"key_col_6_1_1": 100,
				"key_col_6_1_2": 200,
			},
			"key_col_6_2": {
				"key_col_6_2_1": 100,
				"key_col_6_2_2": 200,
			},
		}
	)
	require.NoError(t, batch.Append(col1Data, col2Data, col3Data, col4Data, col5Data, col6Data))
	require.NoError(t, batch.Send())
	var (
		col1 map[string]uint64
		col2 map[string]uint64
		col3 map[string]uint64
		col4 []map[string]string
		col5 map[string]uint64
		col6 map[string]map[string]uint64
	)
	require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_map").Scan(&col1, &col2, &col3, &col4, &col5, &col6))
	assert.Equal(t, col1Data, col1)
	assert.Equal(t, col2Data, col2)
	assert.Equal(t, col3Data, col3)
	assert.Equal(t, col4Data, col4)
	assert.Equal(t, col5Data, col5)
	assert.Equal(t, col6Data, col6)
}

func TestColmnarMap(t *testing.T) {
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
			//Debug: true,
		})
	)
	require.NoError(t, err)
	if err := CheckMinServerVersion(conn, 21, 9, 0); err != nil {
		t.Skip(err.Error())
		return
	}
	const ddl = `
		CREATE TABLE test_map (
			  Col1 Map(String, UInt64)
			, Col2 Map(String, UInt64)
			, Col3 Map(String, UInt64)
		) Engine Memory
		`
	defer func() {
		conn.Exec(ctx, "DROP TABLE test_map")
	}()
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_map")
	require.NoError(t, err)
	var (
		col1Data = []map[string]uint64{}
		col2Data = []map[string]uint64{}
		col3Data = []map[string]uint64{}
	)
	for i := 0; i < 100; i++ {
		col1Data = append(col1Data, map[string]uint64{
			fmt.Sprintf("key_col_1_%d_1", i): uint64(i),
			fmt.Sprintf("key_col_1_%d_2", i): uint64(i),
		})
		col2Data = append(col2Data, map[string]uint64{
			fmt.Sprintf("key_col_2_%d_1", i): uint64(i),
			fmt.Sprintf("key_col_2_%d_2", i): uint64(i),
		})
		col3Data = append(col3Data, map[string]uint64{})
	}
	if err := batch.Column(0).Append(col1Data); !assert.NoError(t, err) {
		return
	}
	if err := batch.Column(1).Append(col2Data); !assert.NoError(t, err) {
		return
	}
	if err := batch.Column(2).Append(col3Data); !assert.NoError(t, err) {
		return
	}
	require.NoError(t, batch.Send())
	{
		var (
			col1     map[string]uint64
			col2     map[string]uint64
			col3     map[string]uint64
			col1Data = map[string]uint64{
				"key_col_1_10_1": 10,
				"key_col_1_10_2": 10,
			}
			col2Data = map[string]uint64{
				"key_col_2_10_1": 10,
				"key_col_2_10_2": 10,
			}
			col3Data = map[string]uint64{}
		)
		require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_map WHERE Col1['key_col_1_10_1'] = $1", 10).Scan(&col1, &col2, &col3))
		assert.Equal(t, col1Data, col1)
		assert.Equal(t, col2Data, col2)
		assert.Equal(t, col3Data, col3)
	}
}

func TestMapFlush(t *testing.T) {
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
			//Debug: true,
		})
	)
	require.NoError(t, err)
	if err := CheckMinServerVersion(conn, 21, 9, 0); err != nil {
		t.Skip(err.Error())
		return
	}
	const ddl = `
		CREATE TABLE test_map_flush (
			  Col1 Map(String, UInt64)
		) Engine Memory
		`
	defer func() {
		conn.Exec(ctx, "DROP TABLE test_map_flush")
	}()
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_map_flush")
	require.NoError(t, err)
	vals := [1000]map[string]uint64{}
	for i := 0; i < 1000; i++ {
		vals[i] = map[string]uint64{
			"i": uint64(i),
		}
		require.NoError(t, batch.Append(vals[i]))
		require.NoError(t, batch.Flush())
	}
	require.NoError(t, batch.Send())
	rows, err := conn.Query(ctx, "SELECT * FROM test_map_flush")
	require.NoError(t, err)
	i := 0
	for rows.Next() {
		var col1 map[string]uint64
		require.NoError(t, rows.Scan(&col1))
		require.Equal(t, vals[i], col1)
		i += 1
	}
	require.Equal(t, 1000, i)
}
