FROM golang:1.12-alpine AS build

RUN apk add git gcc g++

ENV GO111MODULE=on \
    GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=1

RUN go install sigs.k8s.io/kustomize/kustomize/v3 \
    && (cd /; GO111MODULE=on go get github.com/bitnami-labs/sealed-secrets/cmd/kubeseal@master)

COPY . /app
WORKDIR /app

RUN cd plugin/devjoes/v1/secretsealer \
    && go build -buildmode plugin -o SecretSealer.so ./... \
    && chmod +x SecretSealer.so

# RUN cp /go/bin/kustomize /bin/kustomize \
#     && mkdir -p /root/.config/kustomize/ \
#     && ln -s /app/plugin /root/.config/kustomize/plugin
ENTRYPOINT ["sh"]

# # --------------------

# FROM alpine:3.9

# COPY --from=build /go/bin/kustomize /bin/kustomize
# COPY --from=build /app/plugin /root/.config/kustomize/plugin/
# COPY example /root/

# ENTRYPOINT ["sh"]
 #["/bin/kustomize", "build", "/root/", "--enable_alpha_plugins"]
