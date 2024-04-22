VERSION 0.8
FROM ubuntu:24.04
WORKDIR /workdir

deps:
  ENV DEBIAN_FRONTEND=noninteractive
  RUN apt-get update && apt-get install -y \
    build-essential ca-certificates curl git just zip

  # Install GitHub CLI
  ARG GH_CLI_VERSION=2.48.0
  RUN curl -fsSL -o /etc/apt/keyrings/githubcli-archive-keyring.gpg https://cli.github.com/packages/githubcli-archive-keyring.gpg && \
    chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" > /etc/apt/sources.list.d/github-cli.list && \
    apt-get update && apt-get install -y gh="$GH_CLI_VERSION" && \
    gh --version

  ENV GOROOT=/usr/local/lib/go
  ENV GOPATH=/usr/local/go
  ENV PATH=$PATH:$GOROOT/bin:$GOPATH/bin

  ARG GO_VERSION=1.22.2
  RUN mkdir -p "$GOROOT" && \
    curl -fsSL -o - https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz \
    | tar -C "$GOROOT" -xzf - --strip-components=1 go && \
    go version

  COPY .golangci.yml ./
  RUN golangci_lint_version="$(head -n 1 .golangci.yml | tr -d '# ')" && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@"$golangci_lint_version" && \
    golangci-lint --version


# Ensure that the go.mod reflects all used dependencies.
check:
  FROM +deps
  COPY --dir go.mod go.sum app cmd core crypto db vendor web justfile ./
  RUN git init 2>/dev/null && git add go.mod go.sum 2>/dev/null && \
    go mod tidy 2>/dev/null && \
    changes="$(git diff --name-only)" && \
    rm -rf .git && \
    test -z "$changes" || { echo "ERROR: found uncommitted go mod changes" >&2; exit 1; }


lint:
  FROM +check
  # The Git repo is needed here for `golangci-lint --new-from-rev` to work.
  COPY --dir .git ./
  RUN just lint && just lint report
  SAVE ARTIFACT --keep-ts ./golangci-lint*.txt AS LOCAL .


test:
  FROM +check
  RUN just test


build:
  FROM +check
  COPY --dir .git release ./
  RUN git checkout . && git clean -fxd

  LET gitTag="$(git tag --points-at HEAD)"
  ARG EARTHLY_GIT_BRANCH
  IF [ "$EARTHLY_GIT_BRANCH" = "main" ] || [ -n "$gitTag" ]
    RUN NO_CLEANUP=1 just build
    SAVE ARTIFACT --keep-ts ./dist/*.tar.gz AS LOCAL ./dist/
    SAVE ARTIFACT --keep-ts ./dist/*.zip AS LOCAL ./dist/
    # Save entire dir as an artifact, so that binaries can be reused by the OCI
    # image build steps.
    SAVE ARTIFACT --keep-ts ./dist

    RUN find ./dist -mindepth 1 -type d -printf '%p\0' | xargs -0 rm -rf && \
      cd dist && sha256sum * | tee "disco-$(git describe --tags --abbrev=10 --always --dirty)-checksums.txt"
    SAVE ARTIFACT --keep-ts ./dist/disco-*-checksums.txt AS LOCAL ./dist/
  END


build-oci-linux-amd64:
  BUILD +build

  FROM DOCKERFILE --platform=linux/amd64 .
  COPY +build/dist/disco-*-linux-amd64/disco /usr/local/bin/
  ENTRYPOINT ["/usr/local/bin/disco"]


build-oci-linux-arm64:
  BUILD +build

  FROM DOCKERFILE --platform=linux/arm64 .
  COPY +build/dist/disco-*-linux-arm64/disco /usr/local/bin/
  ENTRYPOINT ["/usr/local/bin/disco"]


build-oci:
  BUILD +build
  FROM +build
  ARG tags="latest"
  ARG latest=""

  FOR tag IN "$tags"
    FROM +build-oci-linux-amd64
    SAVE IMAGE --push hackfixme/disco:"$tag"
    FROM +build-oci-linux-arm64
    SAVE IMAGE --push hackfixme/disco:"$tag"
  END
  FROM busybox
  IF [ -n "$latest" ]
    FROM +build-oci-linux-amd64
    SAVE IMAGE --push hackfixme/disco:latest
    FROM +build-oci-linux-arm64
    SAVE IMAGE --push hackfixme/disco:latest
  END


publish-oci:
  BUILD +build
  FROM +build
  # Publish a `main` tag on every commit to `main`, but if the commit is
  # tagged, also publish it as the `latest` tag.
  LET gitTag="$(git tag --points-at HEAD | sed 's:^v::')"
  LET latest=""
  IF [ -n "$gitTag" ]
    SET latest="true"
  END
  ARG EARTHLY_GIT_BRANCH
  IF [ "$EARTHLY_GIT_BRANCH" = "main" ] || [ -n "$gitTag" ]
    BUILD +build-oci --tags="main $gitTag" --latest="$latest"
  END


publish-gh:
  BUILD +build
  FROM +build

  LET gitTag="$(git tag --points-at HEAD)"
  IF [ -n "$gitTag" ]
    ARG EARTHLY_GIT_HASH
    RUN --push --secret GH_TOKEN /bin/bash -c '
      set -x
      _rel_notes="./release/notes/${gitTag}.md"
      if [ ! -r "$_rel_notes" ] || [ ! -s "$_rel_notes" ]; then
        echo "ERROR: release notes file ${_rel_notes} does not exist!" >&2
        exit 1
      fi
      assets=()
      for asset in ./dist/*; do
        assets+=("$asset")
      done
      gh release create "$gitTag" "${assets[@]}" \
        --target "$EARTHLY_GIT_HASH" -F "$_rel_notes"
    '
  END


publish:
  ARG EARTHLY_CI
  IF [ "$EARTHLY_CI" = "true" ]
    BUILD +publish-oci
    BUILD +publish-gh
  END
