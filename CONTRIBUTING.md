# Contributing to the Atlas CLI plugin for Terraform's MongoDB Atlas Provider

Thank you for your interest in contributing! This project welcomes contributions from the community. Please follow the guidelines below to get started, build, and contribute effectively. For compliance and release processes, see the sections below.

## Building

You can build the binary plugin by running `make build`. You'll need to have Go installed. Then you can run directly the generated binary `./bin/binary terraform [command]` to test your changes.

## Using the plugin from the CLI

You can also use the plugin with your changes from the CLI by running: `make local` and following the instructions displayed.

## Third Party Dependencies and Vulnerability Scanning

We scan our dependencies for vulnerabilities and incompatible licenses using [Snyk](https://snyk.io/).
To run Snyk locally please follow their [CLI reference](https://support.snyk.io/hc/en-us/articles/360003812458-Getting-started-with-the-CLI).

We also use Kondukto to scan for third-party dependency vulnerabilities. Kondukto creates tickets in MongoDB's issue tracking system for any vulnerabilities found.

### SBOM and Compliance
We generate Software Bill of Materials (SBOM) files for each release as part of MongoDB's SSDLC initiative. SBOM Lite files are automatically generated and included as release artifacts. Compliance reports are generated after each release and stored in the compliance/<release-version> directory.

Augmented SBOMs can be generated on customer request for any released version. This can only be done by MongoDB employees as it requires access to our GitHub workflow.

### Papertrail Integration
All releases are recorded using a MongoDB-internal application called Papertrail. This records various pieces of information about releases, including the date and time of the release, who triggered the release (by pushing to Evergreen), and a checksum of each release file.

This is done automatically as part of the release.

### Release Artifact Signing
All releases are signed automatically as part of the release process.