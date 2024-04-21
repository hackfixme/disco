# Disco 🪩🕷️

Disco is a distributed configuration and secrets manager.

It allows securely storing and retrieving arbitrary data locally, and serving it to authorized remote clients.

> [!WARNING]
> The project is in early development, so expect bugs and missing features.
> It is usable in its current state, but please report any issues [here](https://github.com/hackfixme/disco/issues).


## Features

- User-friendly command-line interface.
- Single-binary deployments.
- Data is encrypted at rest and in transit using modern cryptography (NaCl, TLS 1.3).
- Flexible authorization using role-based access control.
- Namespacing support for separating environments (development, staging, production, etc.).
- Cross-platform: runs on Linux, macOS and Windows.

You can see planned work on the [roadmap](https://github.com/orgs/hackfixme/projects/1/views/1). Please vote on issues by giving them a :thumbsup:.


## Installation

The easiest way to install Disco is by downloading one of the pre-built packages from the latest release on the [releases page](https://github.com/hackfixme/disco/releases). Unpack the `disco` binary and place it somewhere on your `$PATH`.

Container images are also available on [Docker Hub](https://hub.docker.com/r/hackfixme/disco). Stable releases are published with version tags, e.g. `0.1.0`, and are also tagged with `latest`. Unstable releases track the `main` branch and are tagged with `main`.

To pull and run the latest stable release:
```sh
docker run --rm -it hackfixme/disco --version
```

Alternatively, you can build a binary for your system.

First, ensure [Git](https://github.com/git-guides/install-git) and [Go](https://golang.org/doc/install) are installed. Go must be version 1.22 and later.

Then in a terminal run:

```sh
go install go.hackfix.me/disco/cmd/disco@latest
```


## Documentation

For usage instructions, details about the internals, and other information, see the [documentation](/docs).


## License

[AGPLv3](/LICENSE.md)
