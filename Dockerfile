FROM golang:1.21 as build

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
COPY crawler ./crawler
COPY parser ./parser
COPY scheduler ./scheduler

RUN CGO_ENABLED=0 GOOS=linux go build -o web-crawler

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /workspace/web-crawler .
USER nonroot:nonroot
ENTRYPOINT ["/web-crawler"]
