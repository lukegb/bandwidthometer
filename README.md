Since I now live in easy walking distance of university, the bus-o-meter was less necessary (aww).

Instead, I decided to see when my flatmates were being bandwidth hogs, by introducing...

The Bandwidth-o-meter!

# The tech

The components are pretty simple:

* Netgear R7000 (my router, has an ARMv7 core so is easy to build for in Go)
* The associated `server` code, written in Go
* A Particle Core (because I have one, and wifi is useful)
* The associated `particle` code, written in Arduino-style C (with the `loop` and `setup` functions)
* A WS2812B-compatible strip of 150 LEDs.

# The setup

This basically just took me a morning to build. On the server side, I read out of
`/sys/class/net/*interface*/statistics/rx_bytes` and 
`/sys/class/net/*interface*/statistics/tx_bytes` once a second and compute a delta.

I then ship this delta off, encoded as two big-endian int32s, over broadcast UDP on port 31814
(which, although spammy as all hell, isn't too bad, since it's only once a second...)

The Particle is then listening for this broadcast and picks it up.

# Using this yourself

If you want to use this yourself, you'll need to change a couple things across two places:

* The interface (`server/server.go`) - you need to change "vlan2" to be the interface you want to monitor. In my case, vlan2 is the WAN.
* The maximum RX bandwidth (`particle/bandwidthmeter.ino`) - `RX_BANDWIDTH_MAX` is the maximum receive bandwidth of that interface in *bytes* per second.
* The maximum TX bandwidth (`particle/bandwidthmeter.ino`) - `TX_BANDWIDTH_MAX` is the maximum transmit bandwidth of that interface in *bytes* per second.
* (Maybe) the specific configuration of the LED strip (`particle/bandwidthmeter.ino`) - `PIXEL_COUNT`, `PIXEL_PIN` and `PIXEL_TYPE` should be fairly self-explanatory.
* Potentially the update interval (`server/server.go`) - you need to change `StatsInterval` to something else.

If you want, you can also change the colours. These are declared in `particle/bandwidthmeter.ino` as `RX_COLOUR`, `TX_COLOUR` and `RXTX_COLOUR`. If you want
to change the "standby" colour (i.e. an LED that represents neither RX nor TX activity), then you can also change `NONE_COLOUR`.

# Considerations

* Everyone on your network will be able to see the total bandwidth in use. If you consider this confidential, don't use this. (I plan on also writing a desktop app to expose to myself and my flatmates more easily what the current network saturation is like - but not who is causing it!)
* It sends a UDP broadcast once a second. I don't think this interval is particularly problematic and it's only 8 bytes.

# Building

My build process is roughly:

* `GOARCH=arm GOOS=linux go build -o bandwidthd server/server.go && scp bandwidthd 172.27.27.1:bandwidthd`
* `particle flash <my particle id> particle`

Note that the Particle Core is fairly unstable if it's receiving UDP traffic while the flashing is happening. You may wish to stop the server portion
before attempting to flash the particle, or you might find your updates won't take.
