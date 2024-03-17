FROM golang:1.22.0-bookworm

ENV CGO_ENABLED=0
ENV GO111MODULE=on
ENV PACKAGES="ca-certificates git curl bash zsh"
ENV ROOT /app

RUN apt-get update && apt-get install -y ${PACKAGES} && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR ${ROOT}

COPY ./ ./

RUN go mod download

CMD [ "tail", "-f", "/dev/null" ]
