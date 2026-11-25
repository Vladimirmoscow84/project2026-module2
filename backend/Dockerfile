FROM golang@sha256:648f440f42a0958804efb24df176f806f9d353b41f1c0627f666428e40310f6b AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api . && CGO_ENABLED=0 go build -o /out/probe ./cmd/probe
FROM debian@sha256:88200866dfff7ea7f5cbcb6ec7c8a701889efe6fe859fe64d6990e4b07ea4171
WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=build /out/probe /usr/local/bin/probe
USER 65532:65532
ENTRYPOINT ["/app/api"]
