FROM golang:alpine AS build

ENV GO111MODULE=on \
    GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=1


RUN apk add git gcc g++ bash \
    && mkdir -p /root/.config/kustomize/plugin/devjoes/v1/secretsealer/ \
    && go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4

COPY plugin /src/plugin
COPY go.mod /src/go.mod
WORKDIR /src/

RUN go build -buildmode plugin -o /root/.config/kustomize/plugin/devjoes/v1/secretsealer/SecretSealer.so ./plugin/devjoes/v1/secretsealer/SecretSealer.go \
     && go test ./plugin/devjoes/v1/secretsealer/SecretSealer_test.go \
     && chmod +x /root/.config/kustomize/plugin/devjoes/v1/secretsealer/SecretSealer.so

COPY example /src/example
WORKDIR /src/example

RUN ["/bin/bash", "test.sh" ]

FROM alpine AS final
COPY --from=build /root/.config /root/.config
COPY --from=build /go/bin/kustomize /bin/kustomize

CMD ["bash"]