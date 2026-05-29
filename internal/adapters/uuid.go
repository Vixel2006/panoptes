package adapter

import "github.com/google/uuid"

type UUIDGenerator struct{}

func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

func (g *UUIDGenerator) New() string {
	return uuid.New().String()
}
