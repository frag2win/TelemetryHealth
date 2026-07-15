# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added

- Add deterministic failure injection simulator and POST /api/v1/tenant/{tenant_id}/simulate endpoint

- Add interactive service topology graph, nested Gantt waterfalls, YAML sandboxes, and impact simulators ([e0eba10](https://github.com/frag2win/TelemetryHealth/commit/e0eba100a99dc309956719af2d1e3afc6e74e836))

- Overhaul dashboard UI/UX with dynamic selectors, animations, interactive accordions, syntax highlighting, and diff views ([01f34d5](https://github.com/frag2win/TelemetryHealth/commit/01f34d5c72ab3196624094bf90b63e6766e7e2c5))

- Add AnimatedHealthGauge, AI Agent Traces view, and Dark Mode polish ([bb127fa](https://github.com/frag2win/TelemetryHealth/commit/bb127fa0fb7bbbcb624ad5aea32caa86f10f13b0))

- Add OpenAPI/Swagger specs and UI to REST API ([2dab09b](https://github.com/frag2win/TelemetryHealth/commit/2dab09b44fce3bf911adfcf148f964c0041fc20c))

- Add Prometheus /metrics endpoint to control-plane services ([4532f2f](https://github.com/frag2win/TelemetryHealth/commit/4532f2fcb68d2d09348eeb9fdc2bb62fd497c8ee))

- Add cost-optimized single-node AWS Terraform deployment ([e19f0a5](https://github.com/frag2win/TelemetryHealth/commit/e19f0a526ce66d3a6777b325325b836213a1dbef))

### Fixed

- Fix #root flex layout for React integration ([08105bd](https://github.com/frag2win/TelemetryHealth/commit/08105bdab67439a468a665cfe9a15d6db06e59e3))

### Changed

- Complete rewrite of dashboard implementing useTenantData hook, AbortController, proxy relative paths, and resolving all audit bugs ([2a0259b](https://github.com/frag2win/TelemetryHealth/commit/2a0259b2ae061a26151a9947761737108ec2a435))

### Internal

- Remove committed .exe binaries and add .gitignore ([7c222e3](https://github.com/frag2win/TelemetryHealth/commit/7c222e38c2d36789575a94bfbca3dbbd82b740f8))

- Fix github actions permissions for docs-bot ([9f64431](https://github.com/frag2win/TelemetryHealth/commit/9f64431ffbef197b19f8363510c1dd43a268d9ed))
