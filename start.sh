#!/bin/sh -e

./bootstrap $@
vault agent -config=/tmp/vault-agent-config.hcl -log-level=debug
