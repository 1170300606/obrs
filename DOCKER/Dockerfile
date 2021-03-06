FROM alpine:3.9
LABEL maintainer="yusfko@hotmail.com"

# Tendermint will be looking for the genesis file in /tendermint/config/genesis.json
# (unless you change `genesis_file` in config.toml). You can put your config.toml and
# private validator file into /tendermint/config.
#
# The /tendermint/data dir is used by tendermint to store state.
ENV BFTHOME /chain_bft

# OS environment setup
# Set user right away for determinism, create directory for persistence and give our user ownership
# jq and curl used for extracting `pub_key` from private validator while
# deploying tendermint with Kubernetes. It is nice to have bash so the users
# could execute bash commands.

RUN apk update && \
    apk upgrade && \
    apk --no-cache add curl jq bash && \
    addgroup -g 1000 -S tmuser && \
    adduser -u 100 -S -G tmuser tmuser sudoRUN -h "$BFTHOME"

# Run the container with tmuser by default. (UID=100, GID=1000)
USER 100:1000

# Expose the data directory as a volume since there's mutable state in there
VOLUME [ $BFTHOME ]

WORKDIR $BFTHOME

# p2p and rpc port
EXPOSE 26656 26657

ENTRYPOINT ["/usr/bin/chain_bft"]
CMD ["start"]
STOPSIGNAL SIGTERM

ARG BINARY=chain_bft
COPY $BINARY /usr/bin/chain_bft
COPY smallbank.db /root/.chainbft/data/state.db
