FROM gcr.io/distroless/static-debian13
COPY bin/rie-proxy ./rie-proxy
ENTRYPOINT [ "./rie-proxy" ]
