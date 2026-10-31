package grpcserver

import "github.com/google/uuid"

func parseUUID(raw string) (uuid.UUID, error) {
	return uuid.Parse(raw)
}
