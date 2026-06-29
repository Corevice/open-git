package crypto

// ActionSecretEncryptor envelope-encrypts action secret values at rest.
type ActionSecretEncryptor = SecretEncryptor

func NewActionSecretEncryptor(key []byte) *ActionSecretEncryptor {
	return NewSecretEncryptor(key)
}
