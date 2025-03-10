package tests

import (
	"context"
	"database/sql"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
)

func TestSimpleFloat(t *testing.T) {
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
		CREATE TABLE test_float (
			  Col1 Float32,
			  Col2 Float64,
			  Col3 Nullable(Float64)
		) Engine Memory
		`
	defer func() {
		conn.Exec(ctx, "DROP TABLE test_float")
	}()
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_float")
	require.NoError(t, err)
	require.NoError(t, batch.Append(float32(33.1221), sql.NullFloat64{
		Float64: 34.222,
		Valid:   true,
	}, sql.NullFloat64{
		Float64: 0,
		Valid:   false,
	}))
	assert.NoError(t, batch.Send())
	var (
		col1 float32
		col2 sql.NullFloat64
		col3 sql.NullFloat64
	)
	require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_float").Scan(&col1, &col2, &col3))
	require.Equal(t, float32(33.1221), col1)
	require.Equal(t, sql.NullFloat64{
		Float64: 34.222,
		Valid:   true,
	}, col2)
	require.Equal(t, sql.NullFloat64{
		Float64: 0,
		Valid:   false,
	}, col3)
}

func BenchmarkFloat(b *testing.B) {
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
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		conn.Exec(ctx, "DROP TABLE benchmark_float")
	}()

	if err = conn.Exec(ctx, `CREATE TABLE benchmark_float (Col1 Float32, Col2 Float64) ENGINE = Null`); err != nil {
		b.Fatal(err)
	}

	const rowsInBlock = 10_000_000

	for n := 0; n < b.N; n++ {
		batch, err := conn.PrepareBatch(ctx, "INSERT INTO benchmark_float VALUES")
		if err != nil {
			b.Fatal(err)
		}
		for i := 0; i < rowsInBlock; i++ {
			if err := batch.Append(float32(122.112), 322.111); err != nil {
				b.Fatal(err)
			}
		}
		if err = batch.Send(); err != nil {
			b.Fatal(err)
		}
	}
}

func TestFixedFloatFlush(t *testing.T) {
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
	defer func() {
		conn.Exec(ctx, "DROP TABLE fixed_string_flush")
	}()
	const ddl = `
		CREATE TABLE float_flush (
			  Col1 Float32,
			  Col2 Float64	
		) Engine Memory
		`
	require.NoError(t, conn.Exec(ctx, ddl))
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO float_flush")
	require.NoError(t, err)
	val32s := [1000]float32{}
	val64s := [1000]float64{}
	for i := 0; i < 1000; i++ {
		val32s[i] = rand.Float32()
		val64s[i] = rand.Float64()
		batch.Append(val32s[i], val64s[i])
		batch.Flush()
	}
	batch.Send()
	rows, err := conn.Query(ctx, "SELECT * FROM float_flush")
	require.NoError(t, err)
	i := 0
	for rows.Next() {
		var col1 float32
		var col2 float64
		require.NoError(t, rows.Scan(&col1, &col2))
		require.Equal(t, val32s[i], col1)
		require.Equal(t, val64s[i], col2)
		i += 1
	}
	require.Equal(t, 1000, i)
}
