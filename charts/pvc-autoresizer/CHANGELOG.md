# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

This file itself is based on [Keep a CHANGELOG](https://keepachangelog.com/en/0.3.0/).

## [Unreleased]

## [0.7.0] - 2023-02-10

### Added

- Helm chart | Added controller.podAnnotations to helm chart ([#170](https://github.com/topolvm/pvc-autoresizer/pull/170))

### Changed

- Bump helm/chart-releaser-action from 1.4.1 to 1.5.0 ([#175](https://github.com/topolvm/pvc-autoresizer/pull/175))
- appVersion was changed to 0.6.1.

### Contributors

- @jcortejoso

## [0.6.1] - 2023-01-12

### Changed

- appVersion was changed to 0.6.0.

## [0.6.0] - 2022-12-07

### Added

- add podMonitor ([#149](https://github.com/topolvm/pvc-autoresizer/pull/149))

### Contributors

- @mweibel

## [0.5.0] - 2022-08-19

### Changed
- appVersion was changed to 0.5.0.

## [0.4.0] - 2022-07-04

### Added

- Add support for namespace allow list (#120)

### Changed

- Bump helm/chart-testing-action from 2.0.1 to 2.2.1 (#126)
- Bump helm/chart-releaser-action from 1.2.1 to 1.4.0 (#128)

### Contributors

- @ryanprobus

## [0.3.6] - 2022-04-04

### Changed
- appVersion was changed to 0.3.1.

## [0.3.5] - 2022-03-04

### Changed
- appVersion was changed to 0.3.0.

## [0.3.4] - 2022-02-07

### Changed
- appVersion was changed to 0.2.3.

## [0.3.3] - 2022-01-13

### Changed
- appVersion was changed to 0.2.2.

## [0.3.2] - 2021-11-01

### Changed
- appVersion was changed to 0.2.1.

## [0.3.1] - 2021-10-06

### Added
- Add nodeSelector and tolerations to chart (#69)

### Contributors
- @cmotta2016

## [0.3.0] - 2021-09-09

### Changed
- Change license to Apache License Version 2.0. (#66)

## [0.2.0] - 2021-08-10
- First release.

[Unreleased]: https://github.com/topolvm/pvc-autoresizer/compare/pvc-autoresizer-chart-v0.7.0...HEAD
[0.7.0]: https://github.com/topolvm/pvc-autoresizer/compare/pvc-autoresizer-chart-v0.6.1...pvc-autoresizer-chart-v0.7.0
[0.6.0]: https://github.com/topolvm/pvc-autoresizer/compare/pvc-autoresizer-chart-v0.6.0...pvc-autoresizer-chart-v0.6.1
[0.6.0]: https://github.com/topolvm/pvc-autoresizer/compare/pvc-autoresizer-chart-v0.5.0...pvc-autoresizer-chart-v0.6.0
[0.5.0]: https://github.com/topolvm/pvc-autoresizer/compare/pvc-autoresizer-chart-v0.4.0...pvc-autoresizer-chart-v0.5.0
[0.4.0]: https://github.com/topolvm/pvc-autoresizer/compare/pvc-autoresizer-chart-v0.3.6...pvc-autoresizer-chart-v0.4.0
[0.3.6]: https://github.com/topolvm/pvc-autoresizer/compare/pvc-autoresizer-chart-v0.3.5...pvc-autoresizer-chart-v0.3.6
[0.3.5]: https://github.com/topolvm/pvc-autoresizer/compare/pvc-autoresizer-chart-v0.3.4...pvc-autoresizer-chart-v0.3.5
[0.3.4]: https://github.com/topolvm/pvc-autoresizer/compare/pvc-autoresizer-chart-v0.3.3...pvc-autoresizer-chart-v0.3.4
[0.3.3]: https://github.com/topolvm/pvc-autoresizer/compare/pvc-autoresizer-chart-v0.3.2...pvc-autoresizer-chart-v0.3.3
[0.3.2]: https://github.com/topolvm/pvc-autoresizer/compare/pvc-autoresizer-chart-v0.3.1...pvc-autoresizer-chart-v0.3.2
[0.3.1]: https://github.com/topolvm/pvc-autoresizer/compare/pvc-autoresizer-chart-v0.3.0...pvc-autoresizer-chart-v0.3.1
[0.3.0]: https://github.com/topolvm/pvc-autoresizer/compare/pvc-autoresizer-chart-v0.2.0...pvc-autoresizer-chart-v0.3.0
[0.2.0]: https://github.com/topolvm/pvc-autoresizer/compare/ee8a31ac32b1ad40f0bace32317aa1eee4a8225c...pvc-autoresizer-chart-v0.2.0
