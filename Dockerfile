# Stage 1: Build whisper.cpp
FROM gcc:13 as cpp-builder

WORKDIR /cpp

COPY ./whisper.cpp ./whisper.cpp

RUN apt-get update && apt-get install -y cmake make

WORKDIR /cpp/whisper.cpp

RUN ./models/download-ggml-model.sh base.en

RUN cmake -B build && cmake --build build --config Release

# Stage 2: Build Go application
FROM golang:1.23 as go-builder

WORKDIR /go

COPY ./app ./go_app

WORKDIR /go/go_app

RUN go build -o app .

# Stage 3: Final image
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && apt install libgomp1 && rm -rf /var/lib/apt/lists/*

COPY --from=go-builder /go/go_app/app /usr/local/bin/app

COPY --from=cpp-builder /cpp/whisper.cpp/build/src/libwhisper.so.1 /usr/local/lib/

COPY --from=cpp-builder /cpp/whisper.cpp/build/ggml/src/libggml*.so /usr/local/lib/

COPY --from=cpp-builder /cpp/whisper.cpp/build/bin/whisper-cli /usr/local/bin/whisper-cli

COPY --from=cpp-builder /cpp/whisper.cpp/models/ggml-base.en.bin /models/ggml-base.en.bin

RUN echo "/usr/local/lib" > /etc/ld.so.conf.d/whisper.conf && ldconfig

WORKDIR /app

EXPOSE 3000

ENTRYPOINT ["app"]