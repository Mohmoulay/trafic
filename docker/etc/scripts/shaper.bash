#!/bin/bash
#
# LTE (UE<->SGi LAN)
# - UL [iperf-server]: MAXRATE=50Mbit  LATENCY=92ms MTU=1500 IFACE=eth0 PKTLOSS=0.025%
# - DL [iperf-client]: MAXRATE=100Mbit LATENCY=76ms MTU=1500 PKTLOSS=0.025%

set -eu

IFACE=${IFACE:?"set IFACE to the network interface you want to shape, e.g.: eth0"}
LATENCY=${LATENCY:?"set LATENCY to the upstream network latency, e.g.: 50ms"}
PKTLOSS=${PKTLOSS:?"set PKTLOSS to the percentage of packet lost on the upstream link, e.g.: 0.1%"}
MAXRATE=${MAXRATE:?"set MAXRATE to the maximum upstream throughput rate, e.g.: 40Mbit"}
MTU=${MTU:?"set MTU to the maximum transfer unit, e.g.: 1500"}

# drop any previous setting on the interface
echo ">> Dropping previous settings (if any) on ${IFACE}"
tc qdisc del dev ${IFACE} root || true

# shape upstream bandwidth and latency
echo ">> Setting upstream latency to ${LATENCY} and max throughput rate to ${MAXRATE} on ${IFACE}"
tc qdisc add dev ${IFACE} handle 1: root htb default 11
tc class add dev ${IFACE} parent 1: classid 1:1 htb rate ${MAXRATE} ceil ${MAXRATE}
tc class add dev ${IFACE} parent 1:1 classid 1:11 htb rate ${MAXRATE} ceil ${MAXRATE}
tc qdisc add dev ${IFACE} parent 1:11 handle 10: netem delay ${LATENCY} loss ${PKTLOSS}

# show current configuration
echo ">> Shaping settings on ${IFACE}:"
tc qdisc show dev ${IFACE}

# on docker, only loopback MTU matters
echo ">> Setting MTU to ${MTU} on loopback"
ip link set dev lo mtu ${MTU}
ip link show lo
