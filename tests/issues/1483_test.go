package issues

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	clickhouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIssue1483 verifies that time.Time values used as positional (?), numeric ($1),
// and named (@TS via Named()) query parameters preserve sub-second precision when
// filtering DateTime64 columns.
func TestIssue1483(t *testing.T) {
	conn, err := clickhouse_tests.GetConnectionTCP("issues", clickhouse.Settings{
		"max_execution_time": 60,
	}, nil, &clickhouse.Compression{
		Method: clickhouse.CompressionLZ4,
	})
	require.NoError(t, err)
	if !clickhouse_tests.CheckMinServerServerVersion(conn, 22, 0, 0) {
		t.Skip(fmt.Errorf("unsupported clickhouse version"))
	}

	require.NoError(t, conn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS issue_1483 (
			id   String,
			ts   DateTime64(9)
		) ENGINE = MergeTree
		ORDER BY ts
	`))
	defer func() {
		require.NoError(t, conn.Exec(context.Background(), "DROP TABLE issue_1483"))
	}()

	base := time.Now().Round(time.Second).UTC()
	ts1 := base
	ts2 := base.Add(500 * time.Nanosecond)
	ts3 := base.Add(time.Second)

	batch, err := conn.PrepareBatch(context.Background(), "INSERT INTO issue_1483 (id, ts)")
	require.NoError(t, err)
	require.NoError(t, batch.Append("first", ts1))
	require.NoError(t, batch.Append("second", ts2))
	require.NoError(t, batch.Append("third", ts3))
	require.NoError(t, batch.Send())

	// Positional binding: exact match preserves sub-second precision
	t.Run("positional", func(t *testing.T) {
		rows, err := conn.Query(context.Background(),
			"SELECT id FROM issue_1483 WHERE ts = ?", ts2)
		require.NoError(t, err)
		var ids []string
		for rows.Next() {
			var id string
			require.NoError(t, rows.Scan(&id))
			ids = append(ids, id)
		}
		require.NoError(t, rows.Close())
		require.NoError(t, rows.Err())
		assert.Equal(t, []string{"second"}, ids, "positional binding should match exactly ts2")
	})

	// Numeric binding: range filter preserves sub-second precision
	t.Run("numeric", func(t *testing.T) {
		rows, err := conn.Query(context.Background(),
			"SELECT id FROM issue_1483 WHERE ts > $1 ORDER BY ts ASC", ts1)
		require.NoError(t, err)
		var ids []string
		for rows.Next() {
			var id string
			require.NoError(t, rows.Scan(&id))
			ids = append(ids, id)
		}
		require.NoError(t, rows.Close())
		require.NoError(t, rows.Err())
		assert.Equal(t, []string{"second", "third"}, ids, "numeric binding should return ts2 and ts3")
	})

	// Named binding via Named(): exact match preserves sub-second precision
	t.Run("named", func(t *testing.T) {
		rows, err := conn.Query(context.Background(),
			"SELECT id FROM issue_1483 WHERE ts = @TS", clickhouse.Named("TS", ts2))
		require.NoError(t, err)
		var ids []string
		for rows.Next() {
			var id string
			require.NoError(t, rows.Scan(&id))
			ids = append(ids, id)
		}
		require.NoError(t, rows.Close())
		require.NoError(t, rows.Err())
		assert.Equal(t, []string{"second"}, ids, "Named() binding should match exactly ts2")
	})
}
