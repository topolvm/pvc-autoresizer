# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

This file itself is based on [Keep a CHANGELOG](https://keepachangelog.com/en/0.3.0/).

## [Unreleased]

## [0.4.0] - 2022-07-04

### Added

- add CODEOWNERS (#110)
- Add support for namespace allow list (#120)
- automate adding items to project (#123)
- Update github-actions automatically (#124)

### Changed

- revise CODEOWNERS. (#111)
- generalize curl options (#113)
- Modified to use ghcr.io as a container registry (#114)
- Update e2e topolvm version (#116)
- Bump actions/checkout from 2 to 3 (#125)
- Bump actions/setup-go from 2 to 3 (#127)
- Remove setup-python (#130)

### Fixed

- reconcile: do not resize volume if failed to get inode stats (#121)

### Contributors

- @bells17
- @ryanprobus

## [0.3.1] - 2022-04-04

### Fixed
- Modify to using a pvc capacity for calculate new storage request (#104)
- inodes threshold doc (#105)

### Contributors
- @bells17

## [0.3.0] - 2022-03-04

### Notice

The data types of `pvcautoresizer_success_resize_total`, `pvcautoresizer_failed_resize_total` and
`pvcautoresizer_limit_reached_total` are changed to vector.

### Changed
- Extend metrics to include pvc name (#93)

### Contributors
- @tylerauerbeck

## [0.2.3] - 2022-02-07

### Changed
- Support Kubernetes v1.23 (#92)

### Fixed
- Update example to use preferred storage_limit annotation (#94)

### Contributors
- @tylerauerbeck

## [0.2.2] - 2022-01-12

### Changed
- Support Kubernetes v1.22 (#85)

### Contributors
- @bells17

## [0.2.1] - 2021-11-01

### Added
- Add inode checking feature (#65)
- Storage limit reached (#75)

### Fixed
- output error when storage_limit annotation is invalid (#76)

### Contributors
- @bells17
- @cmotta2016

## [0.2.0] - 2021-09-08
### Changed
- Change license to Apache License Version 2.0.

## [0.1.6] - 2021-08-06

### Added
- Expose metrics (#52, #57)
  - Add metrics description to README.md (#60)
- Add pvc-autoresizer helm charts (#54)

### Changed
- Remove about used_bytes (#49)

### Fixed
- Update kubebuilder to v3 (#41)
- Add e2e test (#44)
- Upgrade controller-runtime to v0.9.2 (#47)
- Add parameter tests for resizing (#48)
- Fix e2e image (#53)

### Contributors
- @bells17
- @d-kuro

## [0.1.5] - 2021-05-06

### Notice

Deprecate specifying an upper limit of volume size with `.spec.resources.limits.storage`.
You can specify the limit by the annotation `resize.topolvm.io/storage_limit`.

### Added
- Add support to providing PVC storage limit via annotation (#32)

### Changed
- don't crash on a single PVC resizing failure (#33)

### Contributors
- @anas-aso

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

[Unreleased]: https://github.com/topolvm/pvc-autoresizer/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/topolvm/pvc-autoresizer/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/topolvm/pvc-autoresizer/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/topolvm/pvc-autoresizer/compare/v0.2.3...v0.3.0
[0.2.3]: https://github.com/topolvm/pvc-autoresizer/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/topolvm/pvc-autoresizer/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/topolvm/pvc-autoresizer/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/topolvm/pvc-autoresizer/compare/v0.1.6...v0.2.0
[0.1.6]: https://github.com/topolvm/pvc-autoresizer/compare/v0.1.5...v0.1.6
[0.1.5]: https://github.com/topolvm/pvc-autoresizer/compare/v0.1.4...v0.1.5
[0.1.4]: https://github.com/topolvm/pvc-autoresizer/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/topolvm/pvc-autoresizer/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/topolvm/pvc-autoresizer/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/topolvm/pvc-autoresizer/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/topolvm/pvc-autoresizer/compare/ee8a31ac32b1ad40f0bace32317aa1eee4a8225c...v0.1.0
