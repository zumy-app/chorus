# =============================================================================
# Translator Engine — llama.cpp server with 7B translation model (GGUF)
# =============================================================================
# Build stage: compile llama.cpp from source with CPU optimizations
FROM alpine:3.21 AS builder

RUN apk add --no-cache \
    build-base \
    cmake \
    curl \
    git \
    linux-headers \
    musl-dev

WORKDIR /build

# Clone llama.cpp at a stable release
ARG LLAMACPP_REV=b4331
RUN git clone --depth 1 --branch "${LLAMACPP_REV}" \
    https://github.com/ggerganov/llama.cpp.git .

# Build server binary statically with CPU-only optimisations (AVX2).
# Disable CUDA and Metal to keep the image lean.
RUN cmake -B build \
    -DBUILD_SHARED_LIBS=OFF \
    -DGGML_CUDA=OFF \
    -DGGML_METAL=OFF \
    -DGGML_NATIVE=ON \
    -DGGML_AVX2=ON \
    -DGGML_AVX512=OFF \
    -DCMAKE_BUILD_TYPE=Release && \
    cmake --build build --config Release -j "$(nproc)" --target llama-server

# Strip the binary
RUN ls -lh build/bin/llama-server && \
    strip build/bin/llama-server && \
    ls -lh build/bin/llama-server

# ---------------------------------------------------------------------------
# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache \
    bash \
    coreutils \
    curl \
    libgomp \
    libstdc++ \
    ca-certificates

# Copy the llama-server binary
COPY --from=builder /build/build/bin/llama-server /usr/local/bin/llama-server

# Copy the startup script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Model directory (use a Docker volume for persistence)
ENV MODEL_DIR=/models

EXPOSE 5000

ENTRYPOINT ["/entrypoint.sh"]
