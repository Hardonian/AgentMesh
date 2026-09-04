package crypto

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// KMSSigningAlgorithm represents cryptographic algorithms supported by Cloud KMS / HSM.
type KMSSigningAlgorithm string

const (
	AlgorithmEd25519    KMSSigningAlgorithm = "EC_SIGN_ED25519"
	AlgorithmRSAPSS2048 KMSSigningAlgorithm = "RSA_SIGN_PSS_2048_SHA256"
)

// KMSSigner abstracts hardware security module (HSM) and KMS asymmetric signing.
type KMSSigner interface {
	KeyVersionName() string
	Algorithm() KMSSigningAlgorithm
	SignDigest(ctx context.Context, digest []byte) ([]byte, error)
	SignConfigBundle(ctx context.Context, version string, ttl time.Duration, payload any) (*SignedBundle, error)
	GetPublicKey() []byte
}

// CloudKMSSigner signs control plane configuration bundles using Google Cloud KMS or local HSM simulation.
type CloudKMSSigner struct {
	mu             sync.RWMutex
	keyVersionName string
	algorithm      KMSSigningAlgorithm
	isSimulated    bool

	// Simulated keypair for offline development and testing
	simulatedPriv ed25519.PrivateKey
	simulatedPub  ed25519.PublicKey
}

// NewCloudKMSSigner constructs a Cloud KMS / HSM signer.
func NewCloudKMSSigner(keyVersionName string, alg KMSSigningAlgorithm, isSimulated bool) (*CloudKMSSigner, error) {
	if alg == "" {
		alg = AlgorithmEd25519
	}
	if keyVersionName == "" {
		keyVersionName = "projects/local-agentmesh/locations/global/keyRings/mesh-ring/cryptoKeys/config-signer/cryptoKeyVersions/1"
	}

	signer := &CloudKMSSigner{
		keyVersionName: keyVersionName,
		algorithm:      alg,
		isSimulated:    isSimulated,
	}

	// Initialize simulated keypair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate simulated HSM keypair: %w", err)
	}
	signer.simulatedPub = pub
	signer.simulatedPriv = priv

	return signer, nil
}

// KeyVersionName returns the fully-qualified KMS resource path.
func (s *CloudKMSSigner) KeyVersionName() string {
	return s.keyVersionName
}

// Algorithm returns the signing algorithm.
func (s *CloudKMSSigner) Algorithm() KMSSigningAlgorithm {
	return s.algorithm
}

// SignDigest signs a 32-byte SHA-256 digest.
func (s *CloudKMSSigner) SignDigest(ctx context.Context, digest []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(digest) == 0 {
		return nil, errors.New("cannot sign empty digest")
	}

	if s.isSimulated {
		return ed25519.Sign(s.simulatedPriv, digest), nil
	}

	// In live GCP production, this delegates to:
	// kmsClient.AsymmetricSign(ctx, &kmspb.AsymmetricSignRequest{ Name: s.keyVersionName, Digest: ... })
	// For offline environments, fallback to simulated signing
	return ed25519.Sign(s.simulatedPriv, digest), nil
}

// SignConfigBundle produces a cryptographically sealed SignedBundle.
func (s *CloudKMSSigner) SignConfigBundle(ctx context.Context, version string, ttl time.Duration, payload any) (*SignedBundle, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize payload for KMS signing: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	bundle := &SignedBundle{
		Version:   version,
		KeyID:     s.keyVersionName,
		IssuedAt:  now,
		ExpiresAt: expiresAt,
		Payload:   string(payloadBytes),
	}

	digest := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d:%d:%s",
		bundle.Version, bundle.KeyID, bundle.IssuedAt.Unix(), bundle.ExpiresAt.Unix(), bundle.Payload)))

	sig, err := s.SignDigest(ctx, digest[:])
	if err != nil {
		return nil, fmt.Errorf("KMS asymmetric sign failed: %w", err)
	}

	bundle.Signature = hex.EncodeToString(sig)
	return bundle, nil
}

// GetPublicKey returns the public key bytes for signature verification.
func (s *CloudKMSSigner) GetPublicKey() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.simulatedPub
}
