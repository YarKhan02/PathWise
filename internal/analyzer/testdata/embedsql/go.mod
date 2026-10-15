// Its own module so the parent build never compiles it; the analyzer loads it
// by path. No external dependencies — database/sql and embed are stdlib.
module embedsql

go 1.26
