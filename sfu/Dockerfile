FROM pionwebrtc/ion-sfu:latest-allrpc

COPY config.toml /config.toml

RUN GRPC_HEALTH_PROBE_VERSION=v0.4.5 && \
    wget -qO/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe

ENTRYPOINT [ "sfu", "-c", "/config.toml", "-jaddr", ":7000", "-gaddr", ":50051", "-maddr", ":80" ]