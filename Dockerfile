############# builder
FROM golang:1.16.3 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-provider-alicloud
COPY . .
RUN make install

############# base
FROM alpine:3.13.4 AS base

############# gardener-extension-provider-alicloud
FROM base AS gardener-extension-provider-alicloud

COPY charts /charts
COPY --from=builder /go/bin/gardener-extension-provider-alicloud /gardener-extension-provider-alicloud
ENTRYPOINT ["/gardener-extension-provider-alicloud"]

############# gardener-extension-validator-alicloud
FROM base AS gardener-extension-validator-alicloud

COPY --from=builder /go/bin/gardener-extension-validator-alicloud /gardener-extension-validator-alicloud
ENTRYPOINT ["/gardener-extension-validator-alicloud"]
