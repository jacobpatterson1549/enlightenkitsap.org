# build the server
FROM golang:1.21-alpine3.18 AS BUILDER
WORKDIR /app
COPY . ./
RUN <<EOT
go generate
go test
CGO_ENABLED=0 go build -o enlightenkitsap
EOT

# copy the server to a minimal build image
FROM scratch
WORKDIR /app
COPY --from=BUILDER /app/enlightenkitsap ./
ENTRYPOINT [ "/app/enlightenkitsap" ]