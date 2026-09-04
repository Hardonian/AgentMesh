# Signed Route Mutation & Cryptographic Configuration Chaining

## Cryptographic Configuration Chaining

Every route mutation generates an immutable, cryptographically signed data-plane configuration bundle:

- `ConfigID`: Globally unique configuration identifier.
- `SequenceVersion`: Monotonically increasing sequence number.
- `PayloadHash`: SHA-256 digest of route weights, fallbacks, and capability ID.
- `PreviousConfigHash`: Cryptographic chain pointer linking to the prior configuration's signature.
- `Signature`: HMAC-SHA256 or asymmetric Ed25519 signature verified by data-plane proxies.

## Offline Survivability & Single-Action Rollback

If the SaaS control plane experiences an outage, data-plane proxies continue operating using cached, signed configurations. Single-action rollback immediately restores the prior verified sequence configuration.
