FROM debian:latest

RUN apt update -y

# Copy bmpc code

ADD . .

# Build MP-SPDZ

RUN apt install -y \
    automake \
    build-essential \
    libboost-dev \
    libboost-thread-dev \
    libntl-dev \
    libsodium-dev \
    libssl-dev \
    libtool \
    m4 \
    python3 \
    texinfo \
    yasm

RUN make -C MP-SPDZ -j 8 tldr
RUN make -C MP-SPDZ -j 8 semi-party.x

# Install Go

RUN apt install -y golang

# Install python dependencies

RUN apt install -y python3-pip

RUN pip3 install pyyaml pwntools matplotlib

# Install misc.

RUN apt -y install tcpdump iproute2
