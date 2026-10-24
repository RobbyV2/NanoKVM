# exit

Turns the USB gadget NIC into an internet uplink for the attached host (the
consumer) whose egress is an exit device that dials in over the web address.
`clients/README.md` describes the wire protocol and the clients.

## The gateway exists only while an exit is attached

The gadget lease names the NanoKVM as router and resolver exactly while an exit
is attached to an enabled slot that uses the gadget NIC. Enabling a slot changes
nothing in the lease. A gateway handed out with nothing behind it is a black hole,
and the gadget NIC's link speed (NCM reports about 426 Mbit/s) makes Windows rank
it above the consumer's Wi-Fi, so the host loses its own internet.

`reconcileGateway` owns the rule. It runs from a two-second poll and from every
watchdog tick. An attach writes the marker, restarts udhcpd and re-enumerates. A
detach that outlasts `gatewayHoldDown` removes the marker, restarts udhcpd and
re-enumerates; a shorter one changes nothing, so a reconnecting client is free.
Disabling a slot whose gateway is out withdraws it at once.

The marker is `/var/run/exit.gadget.route` (tmpfs). `S30rndis` emits
`opt router` and `opt dns` only while it exists. `Init` removes any marker it
finds, including the flash copy older releases kept at
`/etc/kvm/exit/gadget.route`, because no exit can be attached to a process that
is only starting.

## Only a re-enumeration takes a gateway back

Measured on Windows 11 with an NCM gadget: a renewal whose lease lacks the router
option drops the resolver and keeps the default route; a link bounce followed by
a NAK and a fresh lease keeps it; a renewal naming a different router adds it
beside the old one. Only the NIC leaving the bus clears the route. Do not replace
the withdrawal re-enumeration with lease-option changes, link bounces or shorter
leases.

A refused re-enumeration (functionfs session, passthrough holding the UDC, media
busy) leaves the change pending. It is retried every `gatewayRetryInterval`
without restarting udhcpd again and reported as the slot's "re-plug the USB
cable" message until a retry succeeds.

Every rebind recreates the gadget NIC. The transaction or poll that rebinds holds
`opMu`, so the D25 subscriber skips and the caller converges the NIC itself:
`convergeSlot` for enabled slots, `S30rndis start` when none is enabled.

## Exit and consumer are different machines

Running an exit client on the consumer routes its traffic out over the gadget NIC
and back into itself. It loses the uplink it was providing, along with any remote
management of it. Recover by regenerating the token from the NanoKVM.
