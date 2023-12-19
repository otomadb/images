# syntax=docker/dockerfile:1@sha256:ac85f380a63b13dfcefa89046420e1781752bab202122f8f50032edf31be0021

# Builder
FROM golang:1.21.1@sha256:19600fdcae402165dcdab18cb9649540bde6be7274dedb5d205b2f84029fe909 AS builder

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
WORKDIR /build

RUN go install github.com/bufbuild/buf/cmd/buf@latest \
  && go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest \
  && go install google.golang.org/protobuf/cmd/protoc-gen-go@latest \
  && go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
ENV PATH $PATH:/root/.local/bin/:/go/bin/

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN buf generate && go build -o main ./cmd/server

# Runner
# hadolint ignore=DL3006
FROM gcr.io/distroless/static-debian11@sha256:9be3fcc6abeaf985b5ecce59451acbcbb15e7be39472320c538d0d55a0834edc AS runner

WORKDIR /app

COPY --from=builder /build/main /

EXPOSE 38080

CMD ["/main"]
