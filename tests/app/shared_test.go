package app_test

import "fmt"

type notFoundError struct {
	kind string
}

func (e *notFoundError) Error() string {
	return fmt.Sprintf("%s not found", e.kind)
}

type mockIDGenerator struct {
	counter int
}

func (g *mockIDGenerator) New() string {
	g.counter++
	return fmt.Sprintf("id-%d", g.counter)
}
