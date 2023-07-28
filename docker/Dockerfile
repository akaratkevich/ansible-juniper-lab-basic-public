FROM ubuntu:22.04

# Install dependencies
COPY exa.cfg .
COPY route-smash-172.py .
RUN apt update
RUN apt install python3-pip net-tools wget mrtparse vim nano -y && \
    rm -rf /var/lib/apt/lists/* && apt clean
RUN pip install exabgp==4.2.17
