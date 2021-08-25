Release procedure
=================

This document describes how to release a new version of pvc-autoresizer.

Versioning
----------

Follow [semantic versioning 2.0.0][semver] to choose the new version number.

Prepare change log entries
--------------------------

Add notable changes since the last release to [CHANGELOG.md](CHANGELOG.md).
It should look like:

```markdown
(snip)
## [Unreleased]

### Added
- Add a notable feature for users (#35)

### Changed
- Change a behavior affecting users (#33)

### Removed
- Remove a feature, users action required (#39)

### Fixed
- Fix something not affecting users or a minor change (#40)

### Contributors
- @hoge
- @foo

(snip)
```

Bump version
------------

1. Determine a new version number.  Export it as an environment variable:

    ```console
    $ VERSION=1.2.3
    $ export VERSION
    ```

2. Make a branch for the release as follows:

    ```console
    $ git checkout main
    $ git pull
    $ git checkout -b bump-$VERSION
    ```

3. Edit `CHANGELOG.md` for the new version ([example][]).
4. Edit `config/default/kustomization.yaml` and update `newTag` value for the new version.
5. Commit the change and create a pull request:

    ```console
    $ git commit -a -m "Bump version to $VERSION"
    $ git push -u origin bump-$VERSION
    ```

6. Merge the new pull request.
7. Add a new tag and push it as follows:

    ```console
    $ git checkout main
    $ git pull
    $ git tag v$VERSION
    $ git push origin v$VERSION
    ```

Publish GitHub release page
---------------------------

Once a new tag is pushed to GitHub, [GitHub Actions][] automatically
builds a tar archive for the new release, and uploads it to GitHub
releases page.

Visit https://github.com/topolvm/pvc-autoresizer/releases to check
the result.  You may manually edit the page to describe the release.


Release Helm Chart
-----------------

pvc-autoresizer Helm Chart will be released independently from pvc-autoresizer's release.
This will prevent the pvc-autoresizer version from going up just by modifying the Helm Chart.

You must change the version of Chart.yaml when making changes to the Helm Chart. CI fails with lint error when creating a Pull Request without changing the version of Chart.yaml.

```console
$ VERSION=1.2.3
$ export VERSION
$ CHART_VERSION=1.2.3
$ export CHART_VERSION
$ sed -r -i "s/version: [[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+/version: ${CHART_VERSION}/g" charts/pvc-autoresizer/Chart.yaml
$ sed -r -i "s/appVersion: [[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+/appVersion: ${VERSION}/g" charts/pvc-autoresizer/Chart.yaml
$ sed -r -i "s/tag:  # [[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+/tag:  # ${VERSION}/g" charts/pvc-autoresizer/values.yaml
```

When you release the Helm Chart, manually run the GitHub Actions workflow for the release.

https://github.com/topolvm/pvc-autoresizer/actions/workflows/helm-release.yaml

When you run workflow, helm/chart-releaser-action will automatically create a GitHub Release.

[semver]: https://semver.org/spec/v2.0.0.html
[example]: https://github.com/cybozu-go/etcdpasswd/commit/77d95384ac6c97e7f48281eaf23cb94f68867f79
[GitHub Actions]: https://github.com/topolvm/pvc-autoresizer/actions
