############# builder
FROM golang:1.13.8 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-provider-alicloud
COPY . .
RUN make install-requirements && make VERIFY=true all

############# base
FROM alpine:3.11.3 AS base

############# gardener-extension-provider-alicloud
FROM base AS gardener-extension-provider-alicloud

COPY charts /charts
COPY --from=builder /go/bin/gardener-extension-provider-alicloud /gardener-extension-provider-alicloud
ENTRYPOINT ["/gardener-extension-provider-alicloud"]

############# gardener-extension-validator-alicloud
FROM base AS gardener-extension-validator-alicloud

COPY --from=builder /go/bin/gardener-extension-validator-alicloud /gardener-extension-validator-alicloud
ENTRYPOINT ["/gardener-extension-validator-alicloud"]