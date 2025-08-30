FROM ubuntu:noble
ARG TARGETARCH

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get install -y \
    curl \
    && apt-get clean -y && rm -rf /var/lib/apt/lists/*

# install go-task
RUN curl -sSL https://github.com/go-task/task/releases/latest/download/task_linux_$TARGETARCH.tar.gz | \
    tar zx -C /usr/local/bin/ task

LABEL org.opencontainers.image.source=https://github.com/choopm/stdfx
