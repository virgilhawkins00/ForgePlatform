# Contributing to Forge Platform

Thank you for your interest in contributing to Forge Platform! This document provides guidelines and instructions for contributing.

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Pull Request Process](#pull-request-process)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Documentation](#documentation)

## Code of Conduct

This project adheres to a Code of Conduct. By participating, you are expected to uphold this code. Please be respectful and constructive in all interactions.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/forge.git
   cd forge
   ```
3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/forge-platform/forge.git
   ```

## Development Setup

### Prerequisites

- Go 1.24 or later
- Make
- TinyGo (for plugin development)
- Ollama (for AI features)

### Building

```bash
# Install dependencies
make deps

# Build the binary
make build

# Run tests
make test

# Run linter
make lint
```

## Making Changes

1. **Create a branch** for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following our coding standards

3. **Write tests** for new functionality

4. **Run the test suite**:
   ```bash
   make test
   ```

5. **Commit your changes** with a descriptive message:
   ```bash
   git commit -m "feat: add new feature description"
   ```

### Commit Message Format

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `style:` - Code style changes (formatting, etc.)
- `refactor:` - Code refactoring
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks

## Pull Request Process

1. **Update your branch** with the latest upstream changes:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Push your branch** to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

3. **Open a Pull Request** on GitHub

4. **Fill out the PR template** with:
   - Description of changes
   - Related issues
   - Testing performed
   - Screenshots (if applicable)

5. **Address review feedback** promptly

## Coding Standards

### Go Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Run `golangci-lint` before committing
- Keep functions small and focused
- Write descriptive variable and function names

### Architecture Guidelines

- Follow Hexagonal Architecture patterns
- Keep domain logic in `internal/core/domain`
- Define interfaces in `internal/core/ports`
- Implement adapters in `internal/adapters`

### Error Handling

- Always handle errors explicitly
- Use wrapped errors with context
- Log errors at appropriate levels

## Testing

- Write unit tests for all new code
- Aim for >80% code coverage
- Use table-driven tests where appropriate
- Mock external dependencies

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package tests
go test ./internal/core/domain/...
```

## Documentation

- Update README.md for user-facing changes
- Add godoc comments to exported functions
- Update ARCHITECTURE.md for design changes
- Include examples in documentation

## Questions?

Feel free to open an issue for questions or join our community discussions.

Thank you for contributing! 🎉

