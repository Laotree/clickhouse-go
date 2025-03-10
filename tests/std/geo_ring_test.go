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

package std

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/paulmach/orb"
	"github.com/stretchr/testify/assert"
)

func TestStdGeoRing(t *testing.T) {
	ctx := clickhouse.Context(context.Background(), clickhouse.WithSettings(clickhouse.Settings{
		"allow_experimental_geo_types": 1,
	}))
	dsns := map[string]string{"Native": "clickhouse://127.0.0.1:9000", "Http": "http://127.0.0.1:8123"}

	for name, dsn := range dsns {
		t.Run(fmt.Sprintf("%s Protocol", name), func(t *testing.T) {
			if conn, err := sql.Open("clickhouse", dsn); assert.NoError(t, err) {
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
					conn.Exec("DROP TABLE test_geo_ring")
				}()
				if _, err := conn.ExecContext(ctx, ddl); assert.NoError(t, err) {
					scope, err := conn.Begin()
					if !assert.NoError(t, err) {
						return
					}
					if batch, err := scope.Prepare("INSERT INTO test_geo_ring"); assert.NoError(t, err) {
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
						if _, err := batch.Exec(col1Data, col2Data); assert.NoError(t, err) {
							if assert.NoError(t, scope.Commit()) {
								var (
									col1 orb.Ring
									col2 []orb.Ring
								)
								if err := conn.QueryRow("SELECT * FROM test_geo_ring").Scan(&col1, &col2); assert.NoError(t, err) {
									assert.Equal(t, col1Data, col1)
									assert.Equal(t, col2Data, col2)
								}
							}
						}
					}
				}
			}
		},
		)
	}
}
