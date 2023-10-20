Release procedure
=================

This document describes how to release a new version of pvc-autoresizer.

Versioning
----------

Follow [semantic versioning 2.0.0][semver] to choose the new version number.

The format of release notes
---------------------------

In the release procedure for both the app and Helm Chart, the release note is generated automatically,
and then it is edited manually. In this step, PRs should be classified based on [Keep a CHANGELOG](https://keepachangelog.com/en/1.1.0/).

The result should look something like:

```markdown
## What's Changed

### Added

* Add a notable feature for users (#35)

### Changed

* Change a behavior affecting users (#33)

### Removed

* Remove a feature, users action required (#39)

### Fixed

* Fix something not affecting users or a minor change (#40)
```

Bump version
------------

1. Determine a new version number, and define the `VERSION` variable.

    ```console
    VERSION=1.2.3
    ```

2. Make a branch for the release as follows:

    ```console
    git switch main
    git pull
    git switch -c bump-$VERSION
    ```

3. Edit `config/default/kustomization.yaml` and update `newTag` value for the new version.

    ```console
    $ sed -i -s "s/newTag:.*/newTag: ${VERSION}/" config/default/kustomization.yaml
    ```

4. Change `TOPOLVM_VERSION` in `e2e/Makefile` to the latest topolvm chart release tag. (e.g. topolvm-chart-vX.Y.Z)
5. Commit the change and create a pull request:

    ```console
    git commit -a -s -m "Bump version to $VERSION"
    git push -u origin bump-$VERSION
    ```

6. Merge the new pull request.
7. Add a new tag and push it as follows:

    ```console
    git switch main
    git pull
    git tag v$VERSION
    git push origin v$VERSION
    ```

8. Once a new tag is pushed, [GitHub Actions][] automatically
   creates a draft release note for the tagged version,
   builds a tar archive for the new release,
   and attaches it to the release note.
   
   Visit https://github.com/topolvm/pvc-autoresizer/releases to check
   the result. 

9. Edit the auto-generated release note
   and remove PRs which contain changes only to the helm chart.
   Then, publish it.

Release Helm Chart
-----------------

pvc-autoresizer Helm Chart will be released independently from pvc-autoresizer's release.
This will prevent the pvc-autoresizer version from going up just by modifying the Helm Chart.

You must change the version of Chart.yaml when making changes to the Helm Chart. CI fails with lint error when creating a Pull Request without changing the version of Chart.yaml.

1. Determine a new version number.  Export it as an environment variable:

    ```console
    $ APPVERSION=1.2.3
    $ export APPVERSION
    $ CHARTVERSION=1.2.3
    $ export CHARTVERSION
    ```

2. Make a branch for the release as follows:

    ```console
    $ git checkout main
    $ git pull
    $ git checkout -b bump-chart-$CHARTVERSION
    ```

3. Update image and chart versions in files below:

    - charts/pvc-autoresizer/Chart.yaml
    - charts/pvc-autoresizer/values.yaml

    ```console
    $ sed -r -i "s/^version: [[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+/version: ${CHARTVERSION}/g" charts/pvc-autoresizer/Chart.yaml
    $ sed -r -i "s/appVersion: [[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+/appVersion: ${APPVERSION}/g" charts/pvc-autoresizer/Chart.yaml
    $ sed -r -i "s/ghcr.io\/topolvm\/pvc-autoresizer:[[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+/ghcr.io\/topolvm\/pvc-autoresizer:${APPVERSION}/g" charts/pvc-autoresizer/Chart.yaml
    $ sed -r -i "s/tag:  # [[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+/tag:  # ${APPVERSION}/g" charts/pvc-autoresizer/values.yaml
    ```

4. Commit the change and create a pull request:

    ```console
    $ git commit -a -s -m "Bump chart version to $CHARTVERSION"
    $ git push -u origin bump-chart-$CHARTVERSION
    ```

5. Create new pull request and merge it.

6. Manually run the GitHub Actions workflow for the release.

   https://github.com/topolvm/pvc-autoresizer/actions/workflows/helm-release.yaml

   When you run workflow, [helm/chart-releaser-action](https://github.com/helm/chart-releaser-action) will automatically create a GitHub Release.

7. Edit the auto-generated release note as follows:
   1. Select the "Previous tag", which is in the form of "pvc-autoresizer-chart-vX.Y.Z".
   2. Clear the textbox, and click "Generate release notes" button.
   3. Remove PRs which do not contain changes to the helm chart.

[semver]: https://semver.org/spec/v2.0.0.html
[example]: https://github.com/cybozu-go/etcdpasswd/commit/77d95384ac6c97e7f48281eaf23cb94f68867f79
[GitHub Actions]: https://github.com/topolvm/pvc-autoresizer/actions
