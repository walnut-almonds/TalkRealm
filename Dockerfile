FROM gcr.io/distroless/static-debian12

ARG APP

COPY ${APP} /talk-realm/app

WORKDIR /talk-realm

USER nonroot

ENTRYPOINT ["/talk-realm/app"]
