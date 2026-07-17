FROM public.ecr.aws/lambda/provided:al2023
ARG binary
RUN test -n "$binary" && echo $binary
COPY "$binary" ./main
ENTRYPOINT [ "./main" ]
