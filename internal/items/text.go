package items

import (
	"strings"
	"unicode/utf8"
)

// isText reports whether s can be a Postgres text value: valid UTF-8 with no NUL byte. Postgres
// refuses either ("invalid byte sequence for encoding \"UTF8\"", SQLSTATE 22021), so a request
// carrying one is malformed and is answered before the database is asked (AOC-069).
func isText(s string) bool { return utf8.ValidString(s) && strings.IndexByte(s, 0) < 0 }
