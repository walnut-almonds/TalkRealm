FROM gcr.io/distroless/static-debian12

ARG APP

COPY ${APP} /talk-realm/app
COPY web/dist/ /talk-realm/web/dist/
COPY data/words.csv /talk-realm/data/words.csv
COPY data/sentences.csv /talk-realm/data/sentences.csv

WORKDIR /talk-realm

USER nonroot

ENTRYPOINT ["/talk-realm/app"]
