# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

This file itself is based on [Keep a CHANGELOG](https://keepachangelog.com/en/0.3.0/).

## [Unreleased]
- Add support to providing PVC storage limit via annotation (#32)

## [0.1.4] - 2021-03-22
### Changed
- Add --no-annotation-check flag (#29)
- Use go 1.16 (#29)

## [0.1.3] - 2021-01-25
### Changed
- Support k8s 1.19 (#21)
- Go 1.15 and Ubuntu 20.04 (#21)

## [0.1.2] - 2020-10-14

### Changed

- Increase size calculation is now based on the current storage size (#15).
- Fix Deployment manifest (#14).

## [0.1.1] - 2020-10-13

### Added

- Health probes (#11).

### Changed

- Updated manifests (#11).

## [0.1.0] - 2020-08-20

This is the first release.

### Contributors

- @moricho
- @chez-shanpu

[Unreleased]: https://github.com/topolvm/pvc-autoresizer/compare/v0.1.4...HEAD
[0.1.4]: https://github.com/topolvm/pvc-autoresizer/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/topolvm/pvc-autoresizer/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/topolvm/pvc-autoresizer/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/topolvm/pvc-autoresizer/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/topolvm/pvc-autoresizer/compare/ee8a31ac32b1ad40f0bace32317aa1eee4a8225c...v0.1.0
