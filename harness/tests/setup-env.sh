# This would need sudo

# Reduce lo MTU to be the default Ethernet MTU
ip link set dev lo mtu 1500

# Traffic control network emulation properties
tc qdisc del dev lo root netem
tc qdisc add dev lo root netem delay 25ms reorder 10% duplicate 1% loss 2%

# Test emulation is activated
# Expected output: 127.0.0.1: time=50.4 ms
ping -c 3  127.0.0.1


