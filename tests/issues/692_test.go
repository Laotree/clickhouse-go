package issues

import (
	"database/sql"
	"github.com/ClickHouse/clickhouse-go/v2/tests/std"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test692(t *testing.T) {
	conn, err := sql.Open("clickhouse", "clickhouse://127.0.0.1:9000")
	require.NoError(t, err)
	if err := std.CheckMinServerVersion(conn, 21, 9, 0); err != nil {
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
			, Col6 Map(String, Map(String, Int64))
		) Engine Memory
		`
	conn.Exec("DROP TABLE test_map")
	defer func() {
		conn.Exec("DROP TABLE test_map")
	}()

	_, err = conn.Exec(ddl)
	require.NoError(t, err)
	scope, err := conn.Begin()
	require.NoError(t, err)
	batch, err := scope.Prepare("INSERT INTO test_map")
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
			{"A": "B"},
			{"C": "D"},
		}
		col5Data = map[string]uint64{
			"key_col_5_1": 100,
			"key_col_5_2": 200,
		}
		col6Data = map[string]map[string]int64{}
	)
	_, err = batch.Exec(col1Data, col2Data, col3Data, col4Data, col5Data, col6Data)
	require.NoError(t, err)
	require.NoError(t, scope.Commit())
	var (
		col1 interface{}
		col2 map[string]uint64
		col3 map[string]uint64
		col4 []map[string]string
		col5 map[string]uint64
		col6 map[string]map[string]int64
	)
	require.NoError(t, conn.QueryRow("SELECT Col1, Col2, Col3, Col4, Col5, Col6 FROM test_map").Scan(&col1, &col2, &col3, &col4, &col5, &col6))
	assert.Equal(t, col1Data, col1)
	assert.Equal(t, col2Data, col2)
	assert.Equal(t, col3Data, col3)
	assert.Equal(t, col4Data, col4)
	assert.Equal(t, col5Data, col5)
	assert.Equal(t, col6Data, col6)
}
