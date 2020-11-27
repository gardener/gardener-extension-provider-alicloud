############# builder
FROM eu.gcr.io/gardener-project/3rd/golang:1.15.5 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-provider-alicloud
COPY . .
RUN make install

############# base
FROM eu.gcr.io/gardener-project/3rd/alpine:3.12.1 AS base

############# gardener-extension-provider-alicloud
FROM base AS gardener-extension-provider-alicloud

COPY charts /charts
COPY --from=builder /go/bin/gardener-extension-provider-alicloud /gardener-extension-provider-alicloud
ENTRYPOINT ["/gardener-extension-provider-alicloud"]

############# gardener-extension-validator-alicloud
FROM base AS gardener-extension-validator-alicloud

COPY --from=builder /go/bin/gardener-extension-validator-alicloud /gardener-extension-validator-alicloud
ENTRYPOINT ["/gardener-extension-validator-alicloud"]
