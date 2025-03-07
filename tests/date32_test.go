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
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
)

func TestDate32(t *testing.T) {
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
			CREATE TABLE test_date32 (
				  ID   UInt8
				, Col1 Date32
				, Col2 Nullable(Date32)
				, Col3 Array(Date32)
				, Col4 Array(Nullable(Date32))
			) Engine Memory
		`
	defer func() {
		conn.Exec(ctx, "DROP TABLE test_date32")
	}()
	type result struct {
		ColID uint8 `ch:"ID"`
		Col1  time.Time
		Col2  *time.Time
		Col3  []time.Time
		Col4  []*time.Time
	}
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_date32")
	require.NoError(t, err)
	var (
		date1, _ = time.Parse("2006-01-02 15:04:05", "2100-01-01 00:00:00")
		date2, _ = time.Parse("2006-01-02 15:04:05", "1925-01-01 00:00:00")
		date3, _ = time.Parse("2006-01-02 15:04:05", "2283-11-11 00:00:00")
	)
	require.NoError(t, batch.Append(uint8(1), date1, &date2, []time.Time{date2}, []*time.Time{&date2, nil, &date1}))
	require.NoError(t, batch.Append(uint8(2), date2, nil, []time.Time{date1}, []*time.Time{nil, nil, &date2}))
	require.NoError(t, batch.Append(uint8(3), date3, nil, []time.Time{date3}, []*time.Time{nil, nil, &date3}))
	require.NoError(t, batch.Send())
	var (
		result1 result
		result2 result
		result3 result
	)
	require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_date32 WHERE ID = $1", 1).ScanStruct(&result1))
	require.Equal(t, date1, result1.Col1)
	assert.Equal(t, 2100, date1.Year())
	assert.Equal(t, 1, int(date1.Month()))
	assert.Equal(t, 1, date1.Day())
	assert.Equal(t, "UTC", result1.Col1.Location().String())
	assert.Equal(t, date2, *result1.Col2)
	assert.Equal(t, []time.Time{date2}, result1.Col3)
	assert.Equal(t, []*time.Time{&date2, nil, &date1}, result1.Col4)

	require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_date32 WHERE ID = $1", 2).ScanStruct(&result2))
	require.Equal(t, date2, result2.Col1)
	assert.Equal(t, "UTC", result2.Col1.Location().String())
	require.Nil(t, result2.Col2)
	assert.Equal(t, 1925, date2.Year())
	assert.Equal(t, 1, int(date2.Month()))
	assert.Equal(t, 1, date2.Day())
	assert.Equal(t, []time.Time{date1}, result2.Col3)
	assert.Equal(t, []*time.Time{nil, nil, &date2}, result2.Col4)

	require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_date32 WHERE ID = $1", 3).ScanStruct(&result3))
	require.Equal(t, date3, result3.Col1)
	assert.Equal(t, "UTC", result3.Col1.Location().String())
	require.Nil(t, result3.Col2)
	assert.Equal(t, 2283, date3.Year())
	assert.Equal(t, 11, int(date3.Month()))
	assert.Equal(t, 11, date3.Day())
	assert.Equal(t, []time.Time{date3}, result3.Col3)
	assert.Equal(t, []*time.Time{nil, nil, &date3}, result3.Col4)

}

func TestNullableDate32(t *testing.T) {
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
			CREATE TABLE test_date32 (
				  Col1 Date32
				, Col2 Nullable(Date32)
			) Engine Memory
		`
	defer func() {
		conn.Exec(ctx, "DROP TABLE test_date32")
	}()
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_date32")
	require.NoError(t, err)
	date, err := time.Parse("2006-01-02 15:04:05", "2283-11-11 00:00:00")
	require.NoError(t, err)
	require.NoError(t, batch.Append(date, date))
	require.NoError(t, batch.Send())
	var (
		col1 *time.Time
		col2 *time.Time
	)
	require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_date32").Scan(&col1, &col2))
	assert.Equal(t, date, *col1)
	assert.Equal(t, date, *col2)
	require.NoError(t, conn.Exec(ctx, "TRUNCATE TABLE test_date32"))
	batch, err = conn.PrepareBatch(ctx, "INSERT INTO test_date32")
	require.NoError(t, err)
	date, err = time.Parse("2006-01-02 15:04:05", "1925-01-01 00:00:00")
	require.NoError(t, err)
	require.NoError(t, batch.Append(date, nil))
	require.NoError(t, batch.Send())
	col2 = nil
	require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_date32").Scan(&col1, &col2))
	require.Nil(t, col2)
	assert.Equal(t, date, *col1)
	assert.Equal(t, date.Unix(), col1.Unix())
}

func TestColumnarDate32(t *testing.T) {
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
		})
	)
	require.NoError(t, err)
	if err := CheckMinServerVersion(conn, 21, 9, 0); err != nil {
		t.Skip(err.Error())
		return
	}
	const ddl = `
		CREATE TABLE test_date32 (
			  ID   UInt64
			, Col1 Date32
			, Col2 Nullable(Date32)
			, Col3 Array(Date32)
			, Col4 Array(Nullable(Date32))
		) Engine Memory
		`
	defer func() {
		conn.Exec(ctx, "DROP TABLE test_date32")
	}()
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_date32")
	require.NoError(t, err)
	var (
		id       []uint64
		col1Data []time.Time
		col2Data []*time.Time
		col3Data [][]time.Time
		col4Data [][]*time.Time
	)
	var (
		date1, _ = time.Parse("2006-01-02 15:04:05", "2283-11-11 00:00:00")
		date2, _ = time.Parse("2006-01-02 15:04:05", "1925-01-01 00:00:00")
	)
	for i := 0; i < 1000; i++ {
		id = append(id, uint64(i))
		col1Data = append(col1Data, date1)
		if i%2 == 0 {
			col2Data = append(col2Data, &date2)
		} else {
			col2Data = append(col2Data, nil)
		}
		col3Data = append(col3Data, []time.Time{
			date1, date2, date1,
		})
		col4Data = append(col4Data, []*time.Time{
			&date2, nil, &date1,
		})
	}
	{
		if err := batch.Column(0).Append(id); !assert.NoError(t, err) {
			return
		}
		if err := batch.Column(1).Append(col1Data); !assert.NoError(t, err) {
			return
		}
		if err := batch.Column(2).Append(col2Data); !assert.NoError(t, err) {
			return
		}
		if err := batch.Column(3).Append(col3Data); !assert.NoError(t, err) {
			return
		}
		if err := batch.Column(4).Append(col4Data); !assert.NoError(t, err) {
			return
		}
	}
	require.NoError(t, batch.Send())
	var result struct {
		Col1 time.Time
		Col2 *time.Time
		Col3 []time.Time
		Col4 []*time.Time
	}
	require.NoError(t, conn.QueryRow(ctx, "SELECT Col1, Col2, Col3, Col4 FROM test_date32 WHERE ID = $1", 11).ScanStruct(&result))
	require.Nil(t, result.Col2)
	assert.Equal(t, date1, result.Col1)
	assert.Equal(t, []time.Time{date1, date2, date1}, result.Col3)
	assert.Equal(t, []*time.Time{&date2, nil, &date1}, result.Col4)
}

func TestDate32Flush(t *testing.T) {
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
			MaxOpenConns: 1,
		})
	)
	if err := CheckMinServerVersion(conn, 21, 9, 0); err != nil {
		t.Skip(err.Error())
		return
	}
	require.NoError(t, err)
	defer func() {
		conn.Exec(ctx, "DROP TABLE date_32_flush")
	}()
	const ddl = `
		CREATE TABLE date_32_flush (
			  Col1 Date32
		) Engine Memory
		`
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO date_32_flush")
	require.NoError(t, err)
	vals := [1000]time.Time{}
	var now = time.Now()

	for i := 0; i < 1000; i++ {
		vals[i] = now.Add(time.Duration(i) * time.Hour)
		batch.Append(vals[i])
		batch.Flush()
	}
	batch.Send()
	rows, err := conn.Query(ctx, "SELECT * FROM date_32_flush")
	require.NoError(t, err)
	i := 0
	for rows.Next() {
		var col1 time.Time
		require.NoError(t, rows.Scan(&col1))
		assert.Equal(t, vals[i].Format("2016-02-01"), col1.Format("2016-02-01"))
		i += 1
	}
}
