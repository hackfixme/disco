FROM gcr.io/distroless/static-debian12

# NOTE: This file is meant to be used with Earthly and can't be used on its own
# to build Disco images.

USER nonroot

ENV DISCO_DATA_DIR=/opt/disco

WORKDIR $DISCO_DATA_DIR

VOLUME $DISCO_DATA_DIR

EXPOSE 2020
