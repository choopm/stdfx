FROM ubuntu:noble

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get install -y \
    curl \
    && apt-get clean -y && rm -rf /var/lib/apt/lists/*

# install go-task
RUN curl -sSL https://github.com/go-task/task/releases/latest/download/task_linux_amd64.deb -o /tmp/task.deb && \
    dpkg -i /tmp/task.deb && \
    rm /tmp/task.deb

LABEL org.opencontainers.image.source=https://github.com/choopm/stdfx
