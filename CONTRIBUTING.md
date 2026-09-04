# Contributing to AgentMesh

Thank you for your interest in contributing to AgentMesh! We welcome contributions from developers across the AI, networking, cloud, and security ecosystems.

## Development Workflow

1. Fork and clone the repository:

   ```bash
   git clone https://github.com/agentmesh/agentmesh.git
   cd AgentMesh
   ```

2. Run local tests and race detection:

   ```bash
   make test
   make test-race
   ```

3. Build and test local executables:

   ```bash
   make build
   ./bin/agentmesh doctor
   ```

4. Launch the local development control plane:

   ```bash
   make dev
   ```

## Coding Guidelines

- **Go Standard**: All code must conform to standard `gofmt` and `go vet` rules.
- **Error Handling**: Use wrapped errors (`fmt.Errorf("...: %w", err)`). Do not panic on normal inputs.
- **Security Boundaries**: Never log secrets, private keys, or raw prompts. Ensure all new actions are tested against the deterministic policy engine.
- **Invariants**: Any PR touching core policy, routing, or reliability must maintain all 15 Critical Invariants (`tests/invariants_test.go`).
