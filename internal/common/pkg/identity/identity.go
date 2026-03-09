package identity

import "github.com/google/uuid"

// Usamos um wrapper para gerar UUIDs
// o que nos permite trocar a implementação no futuro se necessário
// sem afetar o restante do código.
func NewUUIDV7() string {
	id := uuid.Must(uuid.NewV7())
	return id.String()
}

func IsValidUUID(uuidStr string) bool {
	_, err := uuid.Parse(uuidStr)
	return err == nil
}

func IsNotValidUUID(uuidStr string) bool {
	_, err := uuid.Parse(uuidStr)
	return err != nil
}
