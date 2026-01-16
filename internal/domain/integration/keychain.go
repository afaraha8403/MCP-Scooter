package integration

import (
	"fmt"

	"github.com/danieljoos/wincred"
)

// Keychain handles secure storage of credentials.
type Keychain struct {
	prefix string
}

// NewKeychain creates a new keychain manager.
func NewKeychain(prefix string) *Keychain {
	return &Keychain{prefix: prefix}
}

// SetSecret stores a secret in the Windows Credential Manager.
func (k *Keychain) SetSecret(id, secret string) error {
	cred := wincred.NewGenericCredential(fmt.Sprintf("%s:%s", k.prefix, id))
	cred.CredentialBlob = []byte(secret)
	cred.Persist = wincred.PersistSession
	return cred.Write()
}

// GetSecret retrieves a secret from the Windows Credential Manager.
func (k *Keychain) GetSecret(id string) (string, error) {
	cred, err := wincred.GetGenericCredential(fmt.Sprintf("%s:%s", k.prefix, id))
	if err != nil {
		return "", err
	}
	return string(cred.CredentialBlob), nil
}

// RemoveSecret deletes a secret from the Windows Credential Manager.
func (k *Keychain) RemoveSecret(id string) error {
	cred, err := wincred.GetGenericCredential(fmt.Sprintf("%s:%s", k.prefix, id))
	if err != nil {
		return err
	}
	return cred.Delete()
}
