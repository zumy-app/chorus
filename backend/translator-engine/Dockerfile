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

# Build server binary with CPU-only optimisations (AVX2/AVX512)
# Disable CUDA, Metal, and Vulkan to keep the image lean.
RUN cmake -B build \
    -DLLAMA_CUDA=OFF \
    -DLLAMA_METAL=OFF \
    -DLLAMA_VULKAN=OFF \
    -DLLAMA_NATIVE=ON \
    -DCMAKE_BUILD_TYPE=Release \
    -DLLAMA_STATIC=ON \
    -DLLAMA_AVX2=ON \
    -DLLAMA_AVX512=OFF && \
    cmake --build build --config Release -j "$(nproc)" --target server

# Strip the binary
RUN ls -lh build/bin/server && \
    strip build/bin/server && \
    ls -lh build/bin/server

# ---------------------------------------------------------------------------
# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache \
    bash \
    coreutils \
    curl \
    libgomp \
    ca-certificates

# Copy the llama-server binary
COPY --from=builder /build/build/bin/server /usr/local/bin/llama-server

# Copy the startup script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Model directory (use a Docker volume for persistence)
ENV MODEL_DIR=/models

EXPOSE 5000

ENTRYPOINT ["/entrypoint.sh"]
