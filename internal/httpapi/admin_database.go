package httpapi

import (
	"net/http"
)

// tableStat is one row of admin/get-table-stats.
type tableStat struct {
	Count int64 `json:"count"`
	Size  int64 `json:"size"`
}

// handleAdminTableStats is POST /api/admin/get-table-stats.
//
// `count` is pg_class.reltuples, the planner's estimate, not a COUNT(*): the
// admin page only needs magnitudes, and counting every table would be slow on
// a large instance. A never-analyzed table reports 0.
func (s *Server) handleAdminTableStats(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.pool == nil {
		writeInternalServerError(w)
		return
	}

	rows, err := s.pool.Query(r.Context(), `
SELECT c.relname, GREATEST(c.reltuples, 0)::bigint AS count, pg_total_relation_size(c.oid) AS size
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
  AND c.relkind IN ('r', 'p')
  AND n.nspname !~ '^pg_toast'
ORDER BY c.relname
`)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	defer rows.Close()

	out := map[string]tableStat{}
	for rows.Next() {
		var name string
		var stat tableStat
		if err := rows.Scan(&name, &stat.Count, &stat.Size); err != nil {
			writeInternalServerError(w)
			return
		}
		out[name] = stat
	}
	if err := rows.Err(); err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAdminIndexStats is POST /api/admin/get-index-stats. It mirrors
// pg_indexes, including the index definition so the page can explain what the
// index covers.
func (s *Server) handleAdminIndexStats(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.pool == nil {
		writeInternalServerError(w)
		return
	}

	rows, err := s.pool.Query(r.Context(), `
SELECT schemaname, tablename, indexname, COALESCE(tablespace, ''), indexdef
FROM pg_indexes
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY tablename, indexname
`)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var schema, table, index, tablespace, def string
		if err := rows.Scan(&schema, &table, &index, &tablespace, &def); err != nil {
			writeInternalServerError(w)
			return
		}
		out = append(out, map[string]any{
			"schemaname": schema,
			"tablename":  table,
			"indexname":  index,
			"tablespace": nilIfEmpty(tablespace),
			"indexdef":   def,
		})
	}
	if err := rows.Err(); err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
