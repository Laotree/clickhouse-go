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
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
)

func TestTuple(t *testing.T) {

	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"127.0.0.1:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
		})
	)
	require.NoError(t, err)
	loc, err := time.LoadLocation("Europe/Lisbon")
	require.NoError(t, err)
	localTime := testDate.In(loc)

	if err := CheckMinServerVersion(conn, 21, 9, 0); err != nil {
		t.Skip(err.Error())
		return
	}
	const ddl = `
		CREATE TABLE test_tuple (
			  Col1 Tuple(String, Int64)
			, Col2 Tuple(String, Int8, DateTime('Europe/Lisbon'))
			, Col3 Tuple(name1 DateTime('Europe/Lisbon'), name2 FixedString(2), name3 Map(String, String))
			, Col4 Array(Array( Tuple(String, Int64) ))
			, Col5 Tuple(LowCardinality(String),           Array(LowCardinality(String)))
			, Col6 Tuple(LowCardinality(Nullable(String)), Array(LowCardinality(Nullable(String))))
			, Col7 Tuple(String, Int64)
		) Engine Memory
		`
	defer func() {
		conn.Exec(ctx, "DROP TABLE test_tuple")
	}()
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_tuple")
	require.NoError(t, err)
	var (
		col1Data = []interface{}{"A", int64(42)}
		col2Data = []interface{}{"B", int8(1), localTime.Truncate(time.Second)}
		col3Data = map[string]interface{}{
			"name1": localTime.Truncate(time.Second),
			"name2": "CH",
			"name3": map[string]string{
				"key": "value",
			},
		}
		col4Data = [][][]interface{}{
			[][]interface{}{
				[]interface{}{"Hi", int64(42)},
			},
		}
		col5Data = []interface{}{
			"LCString",
			[]string{"A", "B", "C"},
		}
		str      = "LCString"
		col6Data = []interface{}{
			&str,
			[]*string{&str, nil, &str},
		}
		col7Data = &[]interface{}{"C", int64(42)}
	)
	require.NoError(t, batch.Append(col1Data, col2Data, col3Data, col4Data, col5Data, col6Data, col7Data))
	require.NoError(t, batch.Send())
	var (
		col1 []interface{}
		col2 []interface{}
		// col3 is a named tuple - we can use map
		col3 map[string]interface{}
		col4 [][][]interface{}
		col5 []interface{}
		col6 []interface{}
		col7 []interface{}
	)
	require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_tuple").Scan(&col1, &col2, &col3, &col4, &col5, &col6, &col7))
	assert.NoError(t, err)
	assert.Equal(t, col1Data, col1)
	assert.Equal(t, col2Data, col2)
	assert.JSONEq(t, toJson(col3Data), toJson(col3))
	assert.Equal(t, col4Data, col4)
	assert.Equal(t, col5Data, col5)
	assert.Equal(t, col6Data, col6)
	assert.Equal(t, col7Data, &col7)
}

func TestNamedTupleWithSlice(t *testing.T) {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"127.0.0.1:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
		})
	)
	require.NoError(t, err)
	// https://github.com/ClickHouse/ClickHouse/pull/36544
	if err := CheckMinServerVersion(conn, 22, 5, 0); err != nil {
		t.Skip(err.Error())
		return
	}
	const ddl = "CREATE TABLE test_tuple (Col1 Tuple(name String, `1` Int64)) Engine Memory"

	defer func() {
		conn.Exec(ctx, "DROP TABLE test_tuple")
	}()
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_tuple")
	require.NoError(t, err)
	// this will fail, slices can only be strongly typed if all slice elements are the same type - see TestNamedTupleWithTypedSlice
	require.Error(t, batch.Append([]string{"A", "2"}))
	batch, _ = conn.PrepareBatch(ctx, "INSERT INTO test_tuple")
	var (
		col1Data = []interface{}{"A", int64(42)}
	)
	require.NoError(t, batch.Append(col1Data))
	require.NoError(t, batch.Send())
	var (
		col1 []interface{}
	)
	require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_tuple").Scan(&col1))
	assert.Equal(t, col1Data, col1)
}

func TestNamedTupleWithTypedSlice(t *testing.T) {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"127.0.0.1:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
		})
	)
	require.NoError(t, err)
	// https://github.com/ClickHouse/ClickHouse/pull/36544
	if err := CheckMinServerVersion(conn, 22, 5, 0); err != nil {
		t.Skip(err.Error())
		return
	}
	const ddl = "CREATE TABLE test_tuple (Col1 Tuple(name String, city String), Col2 Int32) Engine Memory"

	defer func() {
		conn.Exec(ctx, "DROP TABLE test_tuple")
	}()
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_tuple")
	require.NoError(t, err)
	var (
		col1Data = []string{"Dale", "Lisbon"}
		name     = "Geoff"
		city     = "Chicago"
		col2Data = []*string{&name, &city}
	)
	require.NoError(t, batch.Append(col1Data, int32(0)))
	require.NoError(t, batch.Append(col2Data, int32(1)))
	require.NoError(t, batch.Send())
	var (
		col1 []string
	)
	require.NoError(t, conn.QueryRow(ctx, "SELECT Col1 FROM test_tuple ORDER BY Col2 ASC").Scan(&col1))
	assert.Equal(t, col1Data, col1)
}

// named tuples work with maps
func TestNamedTupleWithMap(t *testing.T) {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"127.0.0.1:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
		})
	)
	require.NoError(t, err)
	// https://github.com/ClickHouse/ClickHouse/pull/36544
	if err := CheckMinServerVersion(conn, 22, 5, 0); err != nil {
		t.Skip(err.Error())
		return
	}
	const ddl = "CREATE TABLE test_tuple (Col1 Tuple(name String, id Int64)) Engine Memory"

	defer func() {
		conn.Exec(ctx, "DROP TABLE test_tuple")
	}()
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_tuple")
	require.NoError(t, err)
	// this will fail - see TestNamedTupleWithTypedMap as tuple needs to be same type
	require.Error(t, batch.Append(map[string]string{"name": "A", "id": "1"}))
	batch, _ = conn.PrepareBatch(ctx, "INSERT INTO test_tuple")
	col1Data := map[string]interface{}{"name": "A", "id": int64(1)}
	require.NoError(t, batch.Append(col1Data))
	require.NoError(t, batch.Send())
	var (
		col1 map[string]interface{}
	)
	require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_tuple").Scan(&col1))
	assert.Equal(t, col1Data, col1)
}

// named tuples work with typed maps
func TestNamedTupleWithTypedMap(t *testing.T) {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"127.0.0.1:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
		})
	)
	require.NoError(t, err)
	// https://github.com/ClickHouse/ClickHouse/pull/36544
	if err := CheckMinServerVersion(conn, 22, 5, 0); err != nil {
		t.Skip(err.Error())
		return
	}
	const ddl = "CREATE TABLE test_tuple (Col1 Tuple(id Int64, code Int64)) Engine Memory"

	defer func() {
		conn.Exec(ctx, "DROP TABLE test_tuple")
	}()
	// typed maps can be used provided the Tuple is consistent
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_tuple")
	require.NoError(t, err)
	var (
		col1Data = map[string]int64{"code": int64(1), "id": int64(2)}
	)
	require.NoError(t, batch.Append(col1Data))
	require.NoError(t, batch.Send())
	var (
		col1 map[string]int64
	)
	require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_tuple").Scan(&col1))
	assert.Equal(t, col1Data, col1)
}

// test column names which need escaping
func TestNamedTupleWithEscapedColumns(t *testing.T) {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"127.0.0.1:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
		})
	)
	require.NoError(t, err)
	// https://github.com/ClickHouse/ClickHouse/pull/36544
	if err := CheckMinServerVersion(conn, 22, 5, 0); err != nil {
		t.Skip(err.Error())
		return
	}
	const ddl = "CREATE TABLE test_tuple (Col1 Tuple(`56` String, `a22\\`` Int64)) Engine Memory"
	defer func() {
		conn.Exec(ctx, "DROP TABLE test_tuple")
	}()
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_tuple")
	require.NoError(t, err)
	var (
		col1Data = map[string]interface{}{"56": "A", "a22`": int64(1)}
	)
	require.NoError(t, batch.Append(col1Data))
	require.NoError(t, batch.Send())
	var col1 map[string]interface{}
	require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_tuple").Scan(&col1))
	assert.Equal(t, col1Data, col1)
}

func TestNamedTupleIncomplete(t *testing.T) {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"127.0.0.1:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
		})
	)
	require.NoError(t, err)
	// https://github.com/ClickHouse/ClickHouse/pull/36544
	if err := CheckMinServerVersion(conn, 22, 5, 0); err != nil {
		t.Skip(err.Error())
		return
	}
	const ddl = "CREATE TABLE test_tuple (Col1 Tuple(name String, id Int64)) Engine Memory"

	defer func() {
		conn.Exec(ctx, "DROP TABLE test_tuple")
	}()
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_tuple")
	require.NoError(t, err)
	require.Error(t, batch.Append(map[string]interface{}{"name": "A"}))
	require.Error(t, batch.Append([]interface{}{"Dale"}))
}

// unnamed tuples will not work with maps - keys cannot be attributed to fields
func TestUnNamedTupleWithMap(t *testing.T) {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"127.0.0.1:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
		})
	)
	require.NoError(t, err)
	// https://github.com/ClickHouse/ClickHouse/pull/36544
	if err := CheckMinServerVersion(conn, 22, 5, 0); err != nil {
		t.Skip(err.Error())
		return
	}
	const ddl = "CREATE TABLE test_tuple (Col1 Tuple(String, Int64)) Engine Memory"

	defer func() {
		conn.Exec(ctx, "DROP TABLE test_tuple")
	}()
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_tuple")
	require.NoError(t, err)
	var (
		col1Data = map[string]interface{}{"name": "A", "id": int64(1)}
	)
	// this will fail - maps can't be used for unnamed tuples
	err = batch.Append(col1Data)
	require.Error(t, err)
	require.Equal(t, "clickhouse [AppendRow]: (Col1 Tuple(String, Int64)) converting from map[string]interface {} is not supported for unnamed tuples - use a slice", err.Error())
	// insert some data properly to test scan - can't reuse batch
	batch, err = conn.PrepareBatch(ctx, "INSERT INTO test_tuple")
	require.NoError(t, err)
	require.NoError(t, batch.Append([]interface{}{"A", int64(42)}))
	require.NoError(t, batch.Send())
	var col1 map[string]interface{}
	err = conn.QueryRow(ctx, "SELECT * FROM test_tuple").Scan(&col1)
	require.Error(t, err)
	require.Equal(t, "clickhouse [ScanRow]: (Col1) converting Tuple(String, Int64) to map[string]interface {} is unsupported. cannot use maps for unnamed tuples, use slice", err.Error())
}

func TestColumnarTuple(t *testing.T) {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"127.0.0.1:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
		})
	)
	require.NoError(t, err)
	if err := CheckMinServerVersion(conn, 21, 9, 0); err != nil {
		t.Skip(err.Error())
		return
	}
	const ddl = `
		CREATE TABLE test_tuple (
			  ID   UInt64
			, Col1 Tuple(String, Int64)
			, Col2 Tuple(String, Int8, DateTime)
			, Col3 Tuple(DateTime, FixedString(2), Map(String, String))
			, Col4 Tuple(String, Int64)
		) Engine Memory
		`
	defer func() {
		conn.Exec(ctx, "DROP TABLE test_tuple")
	}()
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_tuple")
	require.NoError(t, err)
	var (
		id        []uint64
		col1Data  = [][]interface{}{}
		col2Data  = [][]interface{}{}
		col3Data  = [][]interface{}{}
		col4Data  = []*[]interface{}{}
		timestamp = time.Now().Truncate(time.Second)
	)
	for i := 0; i < 1000; i++ {
		id = append(id, uint64(i))
		col1Data = append(col1Data, []interface{}{
			fmt.Sprintf("A_%d", i), int64(i),
		})
		col2Data = append(col2Data, []interface{}{
			fmt.Sprintf("B_%d", i), int8(1), timestamp,
		})
		col3Data = append(col3Data, []interface{}{
			timestamp, "CH", map[string]string{
				"key": "value",
			},
		})
		col4Data = append(col4Data, &[]interface{}{
			fmt.Sprintf("C_%d", i), int64(i),
		})
	}
	require.NoError(t, batch.Column(0).Append(id))
	require.NoError(t, batch.Column(1).Append(col1Data))
	require.NoError(t, batch.Column(2).Append(col2Data))
	require.NoError(t, batch.Column(3).Append(col3Data))
	require.NoError(t, batch.Column(4).Append(col4Data))
	require.NoError(t, batch.Send())
	{
		var (
			id       uint64
			col1     []interface{}
			col2     []interface{}
			col3     []interface{}
			col4     []interface{}
			col1Data = []interface{}{
				"A_542", int64(542),
			}
			col2Data = []interface{}{
				"B_542", int8(1), timestamp,
			}
			col3Data = []interface{}{
				timestamp, "CH", map[string]string{
					"key": "value",
				},
			}
			col4Data = &[]interface{}{
				"C_542", int64(542),
			}
		)
		require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_tuple WHERE ID = $1", 542).Scan(&id, &col1, &col2, &col3, &col4))
		assert.Equal(t, col1Data, col1)
		assert.Equal(t, col2Data, col2)
		assert.Equal(t, col3Data, col3)
		assert.Equal(t, col4Data, &col4)
	}
}

func TestTupleFlush(t *testing.T) {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"127.0.0.1:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
		})
	)
	require.NoError(t, err)
	if err := CheckMinServerVersion(conn, 21, 9, 0); err != nil {
		t.Skip(err.Error())
		return
	}
	const ddl = `
		CREATE TABLE test_tuple_flush (
			Col1 Tuple(name String, id Int64)
		) Engine Memory
		`
	defer func() {
		conn.Exec(ctx, "DROP TABLE test_tuple_flush")
	}()
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_tuple_flush")
	require.NoError(t, err)
	vals := [1000]map[string]interface{}{}
	for i := 0; i < 1000; i++ {
		vals[i] = map[string]interface{}{
			"id":   int64(i),
			"name": RandAsciiString(10),
		}
		require.NoError(t, batch.Append(vals[i]))
		require.NoError(t, batch.Flush())
	}
	require.NoError(t, batch.Send())
	rows, err := conn.Query(ctx, "SELECT * FROM test_tuple_flush")
	require.NoError(t, err)
	i := 0
	for rows.Next() {
		var col1 map[string]interface{}
		require.NoError(t, rows.Scan(&col1))
		require.Equal(t, vals[i], col1)
		i += 1
	}
	require.Equal(t, 1000, i)

}
