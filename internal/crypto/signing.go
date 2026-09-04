package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// SignedBundle envelopes an arbitrary payload with an Ed25519 signature and freshness metadata.
type SignedBundle struct {
	Version   string    `json:"version"`
	KeyID     string    `json:"keyId"`
	IssuedAt  time.Time `json:"issuedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	Payload   string    `json:"payload"`   // Canonical JSON string
	Signature string    `json:"signature"` // Hex-encoded signature
}

// KeyPair holds an Ed25519 public/private key pair with an identifier.
type KeyPair struct {
	KeyID      string
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// GenerateKeyPair produces a new Ed25519 signing key pair.
func GenerateKeyPair(keyID string) (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ed25519 key pair: %w", err)
	}
	return &KeyPair{
		KeyID:      keyID,
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// SignPayload serializes payload, binds metadata, and signs the digest.
func SignPayload(kp *KeyPair, version string, ttl time.Duration, payloadObj any) (*SignedBundle, error) {
	if kp == nil || kp.PrivateKey == nil {
		return nil, errors.New("cannot sign with nil key")
	}

	payloadBytes, err := json.Marshal(payloadObj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	bundle := &SignedBundle{
		Version:   version,
		KeyID:     kp.KeyID,
		IssuedAt:  now,
		ExpiresAt: expiresAt,
		Payload:   string(payloadBytes),
	}

	digest := computeDigest(bundle.Version, bundle.KeyID, bundle.IssuedAt, bundle.ExpiresAt, bundle.Payload)
	sig := ed25519.Sign(kp.PrivateKey, digest)
	bundle.Signature = hex.EncodeToString(sig)

	return bundle, nil
}

// KeyRing maintains a set of trusted public keys to support seamless key rotation.
type KeyRing struct {
	keys map[string]ed25519.PublicKey
}

// NewKeyRing creates an empty key ring.
func NewKeyRing() *KeyRing {
	return &KeyRing{
		keys: make(map[string]ed25519.PublicKey),
	}
}

// RegisterKey adds a trusted public key to the key ring.
func (kr *KeyRing) RegisterKey(keyID string, pubKey ed25519.PublicKey) {
	kr.keys[keyID] = pubKey
}

// Verify checks the signature, key authenticity, and expiration of the bundle.
func (kr *KeyRing) Verify(bundle *SignedBundle) error {
	if bundle == nil {
		return errors.New("signed bundle is nil")
	}

	pubKey, exists := kr.keys[bundle.KeyID]
	if !exists {
		return fmt.Errorf("unrecognized key ID %q: untrusted signer", bundle.KeyID)
	}

	now := time.Now().UTC()
	if now.After(bundle.ExpiresAt) {
		return fmt.Errorf("bundle has expired at %s (current time: %s)", bundle.ExpiresAt.Format(time.RFC3339), now.Format(time.RFC3339))
	}

	sigBytes, err := hex.DecodeString(bundle.Signature)
	if err != nil {
		return fmt.Errorf("invalid hex signature: %w", err)
	}

	digest := computeDigest(bundle.Version, bundle.KeyID, bundle.IssuedAt, bundle.ExpiresAt, bundle.Payload)
	if !ed25519.Verify(pubKey, digest, sigBytes) {
		return errors.New("cryptographic signature verification failed: tampered payload")
	}

	return nil
}

func computeDigest(version, keyID string, issuedAt, expiresAt time.Time, payload string) []byte {
	raw := fmt.Sprintf("%s|%s|%d|%d|%s",
		version,
		keyID,
		issuedAt.Unix(),
		expiresAt.Unix(),
		payload,
	)
	h := sha256.Sum256([]byte(raw))
	return h[:]
}
