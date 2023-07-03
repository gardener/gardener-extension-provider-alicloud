############# builder
FROM golang:1.19.3 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-provider-alicloud
COPY . .

ARG EFFECTIVE_VERSION

RUN make install EFFECTIVE_VERSION=$EFFECTIVE_VERSION

############# base
FROM gcr.io/distroless/static-debian11:nonroot AS base

############# gardener-extension-provider-alicloud
FROM base AS gardener-extension-provider-alicloud
WORKDIR /

COPY charts /charts
COPY --from=builder /go/bin/gardener-extension-provider-alicloud /gardener-extension-provider-alicloud
ENTRYPOINT ["/gardener-extension-provider-alicloud"]

############# gardener-extension-admission-alicloud
FROM base as gardener-extension-admission-alicloud
WORKDIR /

COPY --from=builder /go/bin/gardener-extension-admission-alicloud /gardener-extension-admission-alicloud
ENTRYPOINT ["/gardener-extension-admission-alicloud"]
