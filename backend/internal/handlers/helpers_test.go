package handlers

import "errors"

// errTestDB is a sentinel error used to exercise handler error branches where a
// service or database call fails.
var errTestDB = errors.New("test database error")
