# See https://docs.earthly.dev/docs/earthfile/features
VERSION --try --raw-output 0.8

PROJECT crossplane-contrib/xprin

ARG --global GO_VERSION=1.24.7
ARG --global E2E_CROSSPLANE_V1=1.20.5
ARG --global E2E_CROSSPLANE_V2=2.1.3

# reviewable checks that a branch is ready for review. Run it before opening a
# pull request. It will catch a lot of the things our CI workflow will catch.
reviewable:
  WAIT
    BUILD +generate
  END
  BUILD +lint
  BUILD +test

# test runs unit tests.
test:
  BUILD +go-test

# lint runs linters.
lint:
  BUILD +go-lint

# build builds xprin for your native OS and architecture.
build:
  ARG USERPLATFORM
  BUILD --platform=$USERPLATFORM +go-build

# multiplatform-build builds xprin for all supported OS and architectures.
multiplatform-build:
  ARG RELEASE_ARTIFACTS=false
  BUILD +go-multiplatform-build --RELEASE_ARTIFACTS=${RELEASE_ARTIFACTS}

# generate runs code generation. To keep builds fast, it doesn't run as part of
# the build target. It's important to run it explicitly when code needs to be
# generated, for example when you update an API type.
generate:
  BUILD +go-generate

# tidy runs go mod tidy to clean up module dependencies. This is separated from
# generate to avoid unnecessary downloads during development when source files
# change but dependencies don't.
tidy:
  BUILD +go-modules-tidy

# e2e runs the end-to-end tests against both Crossplane v1 and v2.
e2e:
  BUILD +e2e-v1
  BUILD +e2e-v2

# go-modules downloads xprin's go modules. It's the base target of most Go
# related target (go-build, etc).
go-modules:
  ARG NATIVEPLATFORM
  FROM --platform=${NATIVEPLATFORM} golang:${GO_VERSION}
  WORKDIR /xprin
  CACHE --id go-build --sharing shared /root/.cache/go-build
  COPY go.mod go.sum ./
  RUN go mod download
  SAVE ARTIFACT go.mod AS LOCAL go.mod
  SAVE ARTIFACT go.sum AS LOCAL go.sum

# go-modules-tidy tidies and verifies go.mod and go.sum.
go-modules-tidy:
  FROM +go-modules
  CACHE --id go-build --sharing shared /root/.cache/go-build
  COPY --dir cmd/ internal/ .
  RUN go mod tidy
  RUN go mod verify
  SAVE ARTIFACT go.mod AS LOCAL go.mod
  SAVE ARTIFACT go.sum AS LOCAL go.sum

go-generate:
  FROM +go-modules
  CACHE --id go-build --sharing shared /root/.cache/go-build
  COPY --dir cmd/ internal/ .
  COPY generate.go .
  RUN go generate -tags 'generate' .
  SAVE ARTIFACT data AS LOCAL data

# go-build builds xprin binaries for your native OS and architecture.
# Set RELEASE_ARTIFACTS=true to output flat release-ready artifacts to _output/release/
go-build:
  ARG EARTHLY_GIT_SHORT_HASH
  ARG EARTHLY_GIT_COMMIT_TIMESTAMP
  ARG XPRIN_VERSION=v0.0.0-${EARTHLY_GIT_COMMIT_TIMESTAMP}-${EARTHLY_GIT_SHORT_HASH}
  ARG TARGETARCH
  ARG TARGETOS
  ARG GOARCH=${TARGETARCH}
  ARG GOOS=${TARGETOS}
  ARG LDFLAGS="-s -w -X=github.com/crossplane-contrib/xprin/internal/version.version=${XPRIN_VERSION}"
  ARG CGO_ENABLED=0
  ARG RELEASE_ARTIFACTS=false
  FROM +go-modules
  LET ext = ""
  IF [ "$GOOS" = "windows" ]
    SET ext = ".exe"
  END
  CACHE --id go-build --sharing shared /root/.cache/go-build
  COPY --dir cmd/ internal/ .
  RUN go build -ldflags="${LDFLAGS}" -o xprin${ext} ./cmd/xprin
  RUN sha256sum xprin${ext} | head -c 64 > xprin${ext}.sha256
  RUN go build -ldflags="${LDFLAGS}" -o xprin-helpers${ext} ./cmd/xprin-helpers
  RUN sha256sum xprin-helpers${ext} | head -c 64 > xprin-helpers${ext}.sha256
  RUN tar -czvf xprin.tar.gz xprin${ext} xprin${ext}.sha256
  RUN sha256sum xprin.tar.gz | head -c 64 > xprin.tar.gz.sha256
  RUN tar -czvf xprin-helpers.tar.gz xprin-helpers${ext} xprin-helpers${ext}.sha256
  RUN sha256sum xprin-helpers.tar.gz | head -c 64 > xprin-helpers.tar.gz.sha256
  IF [ "$RELEASE_ARTIFACTS" = "true" ]
    # Flat structure with arch suffix for releases: _output/release/xprin_linux_amd64
    SAVE ARTIFACT --keep-ts xprin${ext} AS LOCAL _output/release/xprin_${GOOS}_${GOARCH}${ext}
    SAVE ARTIFACT --keep-ts xprin${ext}.sha256 AS LOCAL _output/release/xprin_${GOOS}_${GOARCH}${ext}.sha256
    SAVE ARTIFACT --keep-ts xprin.tar.gz AS LOCAL _output/release/xprin_${GOOS}_${GOARCH}.tar.gz
    SAVE ARTIFACT --keep-ts xprin.tar.gz.sha256 AS LOCAL _output/release/xprin_${GOOS}_${GOARCH}.tar.gz.sha256
    SAVE ARTIFACT --keep-ts xprin-helpers${ext} AS LOCAL _output/release/xprin-helpers_${GOOS}_${GOARCH}${ext}
    SAVE ARTIFACT --keep-ts xprin-helpers${ext}.sha256 AS LOCAL _output/release/xprin-helpers_${GOOS}_${GOARCH}${ext}.sha256
    SAVE ARTIFACT --keep-ts xprin-helpers.tar.gz AS LOCAL _output/release/xprin-helpers_${GOOS}_${GOARCH}.tar.gz
    SAVE ARTIFACT --keep-ts xprin-helpers.tar.gz.sha256 AS LOCAL _output/release/xprin-helpers_${GOOS}_${GOARCH}.tar.gz.sha256
  ELSE
    # Nested structure for local development: _output/bin/linux_amd64/xprin
    SAVE ARTIFACT --keep-ts xprin${ext} AS LOCAL _output/bin/${GOOS}_${GOARCH}/xprin${ext}
    SAVE ARTIFACT --keep-ts xprin${ext}.sha256 AS LOCAL _output/bin/${GOOS}_${GOARCH}/xprin${ext}.sha256
    SAVE ARTIFACT --keep-ts xprin.tar.gz AS LOCAL _output/bundle/${GOOS}_${GOARCH}/xprin.tar.gz
    SAVE ARTIFACT --keep-ts xprin.tar.gz.sha256 AS LOCAL _output/bundle/${GOOS}_${GOARCH}/xprin.tar.gz.sha256
    SAVE ARTIFACT --keep-ts xprin-helpers${ext} AS LOCAL _output/bin/${GOOS}_${GOARCH}/xprin-helpers${ext}
    SAVE ARTIFACT --keep-ts xprin-helpers${ext}.sha256 AS LOCAL _output/bin/${GOOS}_${GOARCH}/xprin-helpers${ext}.sha256
    SAVE ARTIFACT --keep-ts xprin-helpers.tar.gz AS LOCAL _output/bundle/${GOOS}_${GOARCH}/xprin-helpers.tar.gz
    SAVE ARTIFACT --keep-ts xprin-helpers.tar.gz.sha256 AS LOCAL _output/bundle/${GOOS}_${GOARCH}/xprin-helpers.tar.gz.sha256
  END

# go-multiplatform-build builds xprin binaries for all supported OS
# and architectures.
go-multiplatform-build:
  ARG RELEASE_ARTIFACTS=false
  BUILD \
    --platform=linux/amd64 \
    --platform=linux/arm64 \
    --platform=linux/arm \
    --platform=linux/ppc64le \
    --platform=darwin/arm64 \
    --platform=darwin/amd64 \
    --platform=windows/amd64 \
    +go-build --RELEASE_ARTIFACTS=${RELEASE_ARTIFACTS}

# go-test runs Go unit tests.
go-test:
  FROM +go-modules
  CACHE --id go-build --sharing shared /root/.cache/go-build
  COPY --dir cmd/ internal/ .
  RUN go test -covermode=count -coverprofile=coverage.txt ./...
  SAVE ARTIFACT coverage.txt AS LOCAL _output/tests/coverage.txt

# go-lint lints Go code.
go-lint:
  ARG GOLANGCI_LINT_VERSION=v2.5.0
  FROM +go-modules
  # This cache is private because golangci-lint doesn't support concurrent runs.
  CACHE --id go-lint --sharing private /root/.cache/golangci-lint
  CACHE --id go-build --sharing shared /root/.cache/go-build
  RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin ${GOLANGCI_LINT_VERSION}
  COPY .golangci.yml .
  COPY --dir cmd/ internal/ .
  RUN golangci-lint run --fix
  SAVE ARTIFACT cmd AS LOCAL cmd
  SAVE ARTIFACT internal AS LOCAL internal

# crossplane-cli builds the Crossplane CLI binary (cached by Earthly; reused across e2e runs).
# If no CROSSPLANE_VERSION is provided, it will use the latest stable version.
crossplane-cli:
  ARG CROSSPLANE_VERSION
  FROM alpine:3.23
  RUN apk add --no-cache curl
  RUN XP_VERSION="${CROSSPLANE_VERSION:+v${CROSSPLANE_VERSION}}" sh -c 'curl -sL "https://raw.githubusercontent.com/crossplane/crossplane/main/install.sh" | sh'
  RUN mv crossplane /usr/local/bin/crossplane
  SAVE ARTIFACT /usr/local/bin/crossplane

# e2e-run runs e2e tests using a specific Crossplane version.
# If no CROSSPLANE_VERSION is provided, it will use the latest stable version.
# It uses +crossplane-cli artifact so the CLI install is cached.
e2e-run:
  ARG TARGETARCH
  ARG TARGETOS
  ARG GOARCH=${TARGETARCH}
  ARG GOOS=${TARGETOS}
  ARG CROSSPLANE_VERSION
  FROM earthly/dind:alpine-3.20-docker-26.1.5-r0
  COPY +crossplane-cli/crossplane /usr/local/bin/crossplane
  RUN apk add --no-cache bash
  COPY +go-build/xprin .
  COPY --dir examples/ tests/ ./
  RUN chmod +x tests/e2e/scripts/gen-invalid-tests.sh tests/e2e/scripts/run.sh
  WITH DOCKER
    RUN /tests/e2e/scripts/run.sh
  END

# e2e-v1 runs e2e tests against Crossplane v1.
e2e-v1:
  BUILD --build-arg CROSSPLANE_VERSION=$E2E_CROSSPLANE_V1 +e2e-run

# e2e-v2 runs e2e tests against Crossplane v2.
e2e-v2:
  BUILD --build-arg CROSSPLANE_VERSION=$E2E_CROSSPLANE_V2 +e2e-run

# e2e-regen-expected runs v1 and v2 in parallel, merges artifacts, runs cleanup, then exports.
e2e-regen-expected:
  BUILD +e2e-regen-expected-v1
  BUILD +e2e-regen-expected-v2
  FROM alpine:3.23
  RUN apk add --no-cache bash
  WORKDIR /work
  COPY +e2e-regen-expected-v1/expected v1-expected/
  COPY +e2e-regen-expected-v2/expected v2-expected/
  RUN mkdir -p expected && cp -a v1-expected/. expected/ && cp -a v2-expected/. expected/
  COPY --dir tests/e2e/scripts/ ./
  RUN chmod +x scripts/regen-expected.sh
  RUN CLEANUP=true scripts/regen-expected.sh
  SAVE ARTIFACT expected AS LOCAL tests/e2e/expected

# e2e-regen-expected-v1 runs the regen script for Crossplane v1 only; used in parallel with v2 then merged.
# Full target (not BUILD-only) so COPY +e2e-regen-expected-v1/expected works in e2e-regen-expected.
e2e-regen-expected-v1:
  ARG TARGETARCH
  ARG TARGETOS
  ARG GOARCH=${TARGETARCH}
  ARG GOOS=${TARGETOS}
  FROM earthly/dind:alpine-3.20-docker-26.1.5-r0
  COPY (+crossplane-cli/crossplane --CROSSPLANE_VERSION=$E2E_CROSSPLANE_V1) /usr/local/bin/crossplane
  RUN apk add --no-cache bash
  COPY +go-build/xprin .
  COPY --dir examples/ tests/e2e/scripts/ ./
  RUN chmod +x scripts/gen-invalid-tests.sh scripts/regen-expected.sh
  RUN mkdir expected
  WITH DOCKER
    RUN GENERATE=true scripts/regen-expected.sh
  END
  SAVE ARTIFACT expected

# e2e-regen-expected-v2 runs the regen script for Crossplane v2 only; used in parallel with v1 then merged.
e2e-regen-expected-v2:
  ARG TARGETARCH
  ARG TARGETOS
  ARG GOARCH=${TARGETARCH}
  ARG GOOS=${TARGETOS}
  FROM earthly/dind:alpine-3.20-docker-26.1.5-r0
  COPY (+crossplane-cli/crossplane --CROSSPLANE_VERSION=$E2E_CROSSPLANE_V2) /usr/local/bin/crossplane
  RUN apk add --no-cache bash
  COPY +go-build/xprin .
  COPY --dir examples/ tests/e2e/scripts/ ./
  RUN chmod +x scripts/gen-invalid-tests.sh scripts/regen-expected.sh
  RUN mkdir expected
  WITH DOCKER
    RUN GENERATE=true scripts/regen-expected.sh
  END
  SAVE ARTIFACT expected

