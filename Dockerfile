FROM golang:alpine AS build

ENV GO111MODULE=on \
    GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=1


RUN apk add git gcc g++ bash \
    && go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4

WORKDIR /src/
COPY . .

RUN mkdir -p ~/.config/kustomize/plugin/devjoes/v1/secretsealer/ \
     && go build -buildmode plugin -o ~/.config/kustomize/plugin/devjoes/v1/secretsealer/SecretSealer.so ./SecretSealer.go \
     && mkdir ~/sigs.k8s.io/kustomize/plugin -p \
     && go test

COPY example /src/example
WORKDIR /src/example

RUN ["/bin/bash", "test.sh" ]

FROM alpine AS final
COPY --from=build /root/.config /root/.config
COPY --from=build /go/bin/kustomize /bin/kustomize

CMD ["sh"]