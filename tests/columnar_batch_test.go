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
	"database/sql"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
)

func TestColumnarInterface(t *testing.T) {
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
	require.NoError(t, err)
	const ddl = `
			CREATE TABLE test_column_interface (
				    Col1 UInt8
				  , Col2 String
				  , Col3 DateTime
				  , Col4 String
				  , Col5 DateTime
				  , Col6 Int64	
			) Engine Memory
		`
	defer func() {
		conn.Exec(ctx, "DROP TABLE test_column_interface")
	}()
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_column_interface")
	require.NoError(t, err)
	var (
		col1Data    []uint8
		col2Data    []string
		col3Data    []time.Time
		currentTime = time.Now().Truncate(time.Second)
		col4Data    []sql.NullString
		col5Data    []sql.NullTime
		col6Data    []sql.NullInt64
	)
	for i := 0; i < 150; i++ {
		col1Data = append(col1Data, uint8(i))
		col2Data = append(col2Data, fmt.Sprintf("value_%d", i))
		col3Data = append(col3Data, currentTime)
		col4Data = append(col4Data, sql.NullString{String: fmt.Sprintf("value_%d", i), Valid: true})
		col5Data = append(col5Data, sql.NullTime{Time: currentTime, Valid: true})
		col6Data = append(col6Data, sql.NullInt64{Int64: int64(i), Valid: true})
	}
	require.NoError(t, batch.Column(0).Append(col1Data))
	require.NoError(t, batch.Column(1).Append(col2Data))
	require.NoError(t, batch.Column(2).Append(col3Data))
	require.NoError(t, batch.Column(3).Append(col4Data))
	require.NoError(t, batch.Column(4).Append(col5Data))
	require.NoError(t, batch.Column(5).Append(col6Data))
	require.NoError(t, batch.Send())
	var count uint64
	require.NoError(t, conn.QueryRow(ctx, "SELECT COUNT() FROM test_column_interface").Scan(&count))
	require.Equal(t, uint64(150), count)
	rows, err := conn.Query(ctx, "SELECT * FROM test_column_interface WHERE Col1 >= $1 AND Col1 < $2", 10, 30)
	require.NoError(t, err)
	var (
		row uint8 = 10
	)
	iCount := 0
	for rows.Next() {
		var (
			col1 uint8
			col2 string
			col3 time.Time
			col4 sql.NullString
			col5 sql.NullTime
			col6 sql.NullInt64
		)
		require.NoError(t, rows.Scan(&col1, &col2, &col3, &col4, &col5, &col6))
		assert.Equal(t, row, col1)
		assert.Equal(t, fmt.Sprintf("value_%d", row), col2)
		assert.Equal(t, currentTime.Unix(), col3.Unix())
		assert.Equal(t, fmt.Sprintf("value_%d", row), col4.String)
		assert.Equal(t, currentTime, col5.Time)
		assert.Equal(t, int64(row), col6.Int64)
		row++
		iCount++
	}
	rows.Close()
	require.NoError(t, rows.Err())
	assert.Equal(t, 20, iCount)
}

func TestNullableColumnarInterface(t *testing.T) {
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
			//	Debug: true,
		})
	)
	if assert.NoError(t, err) {
		const ddl = `
			CREATE TABLE test_column_interface (
				  Col1 Nullable(UInt8)
				, Col2 Nullable(String)
				, Col3 Nullable(DateTime)
			) Engine Memory
		`
		defer func() {
			conn.Exec(ctx, "DROP TABLE test_column_interface")
		}()
		if err := conn.Exec(ctx, ddl); assert.NoError(t, err) {
			if batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_column_interface"); assert.NoError(t, err) {
				var (
					col1Data    []*uint8
					col2Data    []*string
					col3Data    []*time.Time
					currentTime = time.Now().Truncate(time.Second)
				)
				for i := 0; i < 150; i++ {
					a, b := uint8(i), fmt.Sprintf("value_%d", i)
					{
						col1Data = append(col1Data, &a)
						col2Data = append(col2Data, &b)
						col3Data = append(col3Data, &currentTime)
					}
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
				if assert.NoError(t, batch.Send()) {
					var count uint64
					if err := conn.QueryRow(ctx, "SELECT COUNT() FROM test_column_interface").Scan(&count); assert.NoError(t, err) {
						if assert.Equal(t, uint64(150), count) {
							rows, err := conn.Query(ctx, "SELECT * FROM test_column_interface WHERE Col1 >= $1 AND Col1 < $2", 10, 30)
							if assert.NoError(t, err) {
								var (
									row   uint8 = 10
									count uint64
								)
								for rows.Next() {
									var (
										col1 *uint8
										col2 *string
										col3 *time.Time
									)
									if assert.NoError(t, rows.Scan(&col1, &col2, &col3)) {
										assert.Equal(t, row, *col1)
										assert.Equal(t, fmt.Sprintf("value_%d", row), *col2)
										assert.Equal(t, currentTime.Unix(), col3.Unix())
									}
									row++
									count++
								}
								rows.Close()
								if assert.NoError(t, rows.Err()) {
									assert.Equal(t, uint64(20), count)
								}
							}
						}
					}
				}
			}
			if err := conn.Exec(ctx, "TRUNCATE TABLE test_column_interface"); !assert.NoError(t, err) {
				return
			}
			if batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_column_interface"); assert.NoError(t, err) {
				var (
					col1Data    []*uint8
					col2Data    []*string
					col3Data    []*time.Time
					currentTime = time.Now().Truncate(time.Second)
				)
				for i := 0; i < 150; i++ {
					a, b := uint8(i), fmt.Sprintf("value_%d", i)
					col1Data = append(col1Data, &a)
					switch {
					case i%2 == 0:
						col2Data = append(col2Data, &b)
						col3Data = append(col3Data, &currentTime)
					default:
						col2Data = append(col2Data, nil)
						col3Data = append(col3Data, nil)
					}
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
				if assert.NoError(t, batch.Send()) {
					var count uint64
					if err := conn.QueryRow(ctx, "SELECT COUNT() FROM test_column_interface").Scan(&count); assert.NoError(t, err) {
						if assert.Equal(t, uint64(150), count) {
							rows, err := conn.Query(ctx, "SELECT * FROM test_column_interface WHERE Col1 >= $1 AND Col1 < $2", 10, 30)
							if assert.NoError(t, err) {
								var (
									row   uint8 = 10
									count uint64
								)
								for rows.Next() {
									var (
										col1 *uint8
										col2 *string
										col3 *time.Time
									)
									if assert.NoError(t, rows.Scan(&col1, &col2, &col3)) {
										switch {
										case row%2 == 0:
											assert.Equal(t, row, *col1)
											assert.Equal(t, fmt.Sprintf("value_%d", row), *col2)
											assert.Equal(t, currentTime.Unix(), col3.Unix())
										default:
											if assert.Equal(t, row, *col1) {
												assert.Nil(t, col2)
												assert.Nil(t, col3)
											}
										}
									}
									row++
									count++
								}
								rows.Close()
								if assert.NoError(t, rows.Err()) {
									assert.Equal(t, uint64(20), count)
								}
							}
						}
					}
				}
			}
		}
	}
}
