# See https://docs.earthly.dev/docs/earthfile/features
VERSION --try --raw-output 0.8

PROJECT crossplane-contrib/xprin

ARG --global GO_VERSION=1.24.7
ARG --global E2E_CROSSPLANE_V1=v1.20.4
ARG --global E2E_CROSSPLANE_V2=v2.1.3

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
  BUILD +go-multiplatform-build

# generate runs code generation. To keep builds fast, it doesn't run as part of
# the build target. It's important to run it explicitly when code needs to be
# generated, for example when you update an API type.
generate:
  BUILD +go-modules-tidy

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

# go-build builds xprin binaries for your native OS and architecture.
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
  SAVE ARTIFACT --keep-ts xprin${ext} AS LOCAL _output/bin/${GOOS}_${GOARCH}/xprin${ext}
  SAVE ARTIFACT --keep-ts xprin${ext}.sha256 AS LOCAL _output/bin/${GOOS}_${GOARCH}/xprin${ext}.sha256
  SAVE ARTIFACT --keep-ts xprin.tar.gz AS LOCAL _output/bundle/${GOOS}_${GOARCH}/xprin.tar.gz
  SAVE ARTIFACT --keep-ts xprin.tar.gz.sha256 AS LOCAL _output/bundle/${GOOS}_${GOARCH}/xprin.tar.gz.sha256
  SAVE ARTIFACT --keep-ts xprin-helpers${ext} AS LOCAL _output/bin/${GOOS}_${GOARCH}/xprin-helpers${ext}
  SAVE ARTIFACT --keep-ts xprin-helpers${ext}.sha256 AS LOCAL _output/bin/${GOOS}_${GOARCH}/xprin-helpers${ext}.sha256
  SAVE ARTIFACT --keep-ts xprin-helpers.tar.gz AS LOCAL _output/bundle/${GOOS}_${GOARCH}/xprin-helpers.tar.gz
  SAVE ARTIFACT --keep-ts xprin-helpers.tar.gz.sha256 AS LOCAL _output/bundle/${GOOS}_${GOARCH}/xprin-helpers.tar.gz.sha256

# go-multiplatform-build builds xprin binaries for all supported OS
# and architectures.
go-multiplatform-build:
  BUILD \
    --platform=linux/amd64 \
    --platform=linux/arm64 \
    --platform=linux/arm \
    --platform=linux/ppc64le \
    --platform=darwin/arm64 \
    --platform=darwin/amd64 \
    --platform=windows/amd64 \
    +go-build

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

# crossplane-cli builds the Crossplane CLI binary (cached by Earthly; reused across test-e2e runs).
crossplane-cli:
  ARG CROSSPLANE_VERSION
  FROM alpine:3.20
  RUN apk add --no-cache curl
  RUN XP_VERSION="${CROSSPLANE_VERSION}" sh -c 'curl -sL "https://raw.githubusercontent.com/crossplane/crossplane/main/install.sh" | sh'
  RUN mv crossplane /usr/local/bin/crossplane
  SAVE ARTIFACT /usr/local/bin/crossplane

# test-e2e runs e2e tests (uses +crossplane-cli artifact so the CLI install is cached).
test-e2e:
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
  RUN chmod +x tests/e2e/scripts/run.sh
  WITH DOCKER
    RUN CROSSPLANE_VERSION=${CROSSPLANE_VERSION} /tests/e2e/scripts/run.sh
  END

# test-e2e-v1 runs tests against Crossplane v1.
test-e2e-v1:
  BUILD --build-arg CROSSPLANE_VERSION=$E2E_CROSSPLANE_V1 +test-e2e

# test-e2e-v2 runs tests against Crossplane v2.
test-e2e-v2:
  BUILD --build-arg CROSSPLANE_VERSION=$E2E_CROSSPLANE_V2 +test-e2e

# test-e2e-all runs the e2e tests against v1 and v2 (sequential; for local use; CI uses matrix jobs).
test-e2e-all:
  BUILD +test-e2e-v1
  BUILD +test-e2e-v2

# regen-e2e-expected-v1 runs the regen script for Crossplane v1 only; used in parallel with v2 then merged.
regen-e2e-expected-v1:
  ARG TARGETARCH
  ARG TARGETOS
  ARG GOARCH=${TARGETARCH}
  ARG GOOS=${TARGETOS}
  FROM earthly/dind:alpine-3.20-docker-26.1.5-r0
  COPY (+crossplane-cli/crossplane --CROSSPLANE_VERSION=${E2E_CROSSPLANE_V1}) /usr/local/bin/crossplane
  RUN apk add --no-cache bash
  COPY +go-build/xprin .
  COPY --dir examples/ tests/e2e/scripts/ ./
  RUN chmod +x scripts/regen-expected.sh
  RUN mkdir expected
  WITH DOCKER
    RUN CROSSPLANE_V1=/usr/local/bin/crossplane scripts/regen-expected.sh v1
  END
  SAVE ARTIFACT expected

# regen-e2e-expected-v2 runs the regen script for Crossplane v2 only; used in parallel with v1 then merged.
regen-e2e-expected-v2:
  ARG TARGETARCH
  ARG TARGETOS
  ARG GOARCH=${TARGETARCH}
  ARG GOOS=${TARGETOS}
  FROM earthly/dind:alpine-3.20-docker-26.1.5-r0
  COPY (+crossplane-cli/crossplane --CROSSPLANE_VERSION=${E2E_CROSSPLANE_V2}) /usr/local/bin/crossplane
  RUN apk add --no-cache bash
  COPY +go-build/xprin .
  COPY --dir examples/ tests/e2e/scripts/ ./
  RUN chmod +x scripts/regen-expected.sh
  RUN mkdir expected
  WITH DOCKER
    RUN CROSSPLANE_V2=/usr/local/bin/crossplane scripts/regen-expected.sh v2
  END
  SAVE ARTIFACT expected

# regen-e2e-expected runs v1 and v2 in parallel, merges artifacts, runs cleanup, then exports.
regen-e2e-expected:
  BUILD +regen-e2e-expected-v1
  BUILD +regen-e2e-expected-v2
  FROM alpine:3.20
  RUN apk add --no-cache bash
  WORKDIR /work
  COPY +regen-e2e-expected-v1/expected expected/
  COPY +regen-e2e-expected-v2/expected expected/
  COPY --dir tests/e2e/scripts/ ./
  RUN chmod +x scripts/regen-expected.sh
  RUN ls -la expected
  RUN scripts/regen-expected.sh cleanup
  SAVE ARTIFACT expected AS LOCAL tests/e2e/expected
