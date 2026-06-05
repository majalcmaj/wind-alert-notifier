FROM golang:1.25 AS build
WORKDIR /build

RUN --mount=type=bind,source=.,target=src,ro \
    --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd src && go build -tags lambda.norpc -o /build/ ./...

# Copy artifacts to a clean image
FROM public.ecr.aws/lambda/provided:al2023
COPY --from=build /build/wind-alert-go ./main
ENTRYPOINT [ "./main" ]
