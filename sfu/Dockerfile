FROM pionwebrtc/ion-sfu:latest-allrpc

COPY config.toml /config.toml

ENTRYPOINT [ "sfu", "-c", "/config.toml", "-jaddr", ":7000", "-gaddr", ":50051", "-maddr", ":80" ]