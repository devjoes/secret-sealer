FROM golang:alpine AS build

ENV GO111MODULE=on \
    GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=1

RUN apk add git gcc g++ \
    && go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4

WORKDIR /src/
COPY . .

RUN mkdir -p /root/.config/kustomize/plugin/devjoes/v1/secretsealer/ \
     && go build -buildmode plugin -o /root/.config/kustomize/plugin/devjoes/v1/secretsealer/SecretSealer.so ./SecretSealer.go \
     && mkdir /root/sigs.k8s.io/kustomize/plugin -p

FROM build AS tests
RUN go test

FROM build AS tests-example
RUN apk add bash
COPY example /src/example
WORKDIR /src/example

RUN ["/bin/bash", "test.sh" ]

FROM tests AS final

COPY --from=build /root/.config /~/.config
COPY --from=build /go/bin/kustomize /bin/kustomize

ENTRYPOINT ["kustomize"]