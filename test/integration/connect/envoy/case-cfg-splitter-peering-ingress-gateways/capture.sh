#!/bin/bash

snapshot_envoy_admin localhost:20000 ingress-gateway primary || true
snapshot_envoy_admin localhost:19000 s1 primary || true
snapshot_envoy_admin localhost:19001 s1 alpha || true
snapshot_envoy_admin localhost:19002 s2 alpha || true
snapshot_envoy_admin localhost:19003 mesh-gateway alpha || true