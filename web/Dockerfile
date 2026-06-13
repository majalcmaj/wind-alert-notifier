FROM golang:1.26 AS build
WORKDIR /build

RUN --mount=type=bind,source=.,target=src,ro \
    --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd src/web && go build -tags lambda.norpc -o /build/ .

# Copy artifacts to a clean image
FROM public.ecr.aws/lambda/provided:al2023
COPY --from=build /build/web ./main
ENTRYPOINT [ "./main" ]
