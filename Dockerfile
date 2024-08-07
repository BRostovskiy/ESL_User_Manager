FROM --platform=linux/arm64 golang:1.22-alpine as builder

ARG API_VERSION

WORKDIR /opt
COPY . .

RUN apk --no-cache add ca-certificates && apk --no-cache add --virtual build-dependencies curl
RUN BIN="/usr/local/bin" && \
    VERSION="1.30.1" && \
      curl -sSL \
        "https://github.com/bufbuild/buf/releases/download/v${VERSION}/buf-$(uname -s)-$(uname -m)" \
        -o "${BIN}/buf" && \
      chmod +x "${BIN}/buf"

RUN go get \
        github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
        github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
        google.golang.org/protobuf/cmd/protoc-gen-go \
        google.golang.org/grpc/cmd/protoc-gen-go-grpc

RUN go install \
        github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
        github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
        google.golang.org/protobuf/cmd/protoc-gen-go \
        google.golang.org/grpc/cmd/protoc-gen-go-grpc
RUN buf mod update
RUN buf generate --path internal/servers/grpc/proto/user-manager/v1/

RUN go build -o user_manager -ldflags "-X main.version=$API_VERSION" ./cmd

FROM alpine:latest
COPY --from=builder /opt/user_manager ./user_manager
COPY --from=builder /opt/compose/um_config.yaml /etc/um_config.yaml
CMD ["./user_manager"]