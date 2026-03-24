# CI/CD Infrastructure

This directory contains information about the Continuous Integration and Deployment pipelines.

## GitHub Actions

The actual workflow definitions are located in `.github/workflows/` at the repository root.

- **`build.yml`**: Runs Docker builds and tests.
- **`pypi.yml`**: Publishes the Scanner package to PyPI.
