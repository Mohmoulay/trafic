# Trafic
Prototype of a traffic mix generator based on [iperf3](https://iperf.fr).

## Dependencies

A UNIX or UNIX-like OS

### Go

Follow instructions at https://golang.org/doc/install

Also, a few 3rd party packages *trafic* depends on:
```
go get -u gopkg.in/yaml.v2 github.com/spf13/cobra github.com/spf13/viper github.com/alecthomas/units
```

To install directly using `go get`, you can just

```
go get -t -u github.com/mami-project/trafic
```

`webhook` can also come in handy to automate at least part of the execution of trafic tests. Install it with :
```
go get github.com/adnanh/webhook
```

And check `docker/etc/scripts` for examples

### Iperf3

Fetch the latest release (>= 3.5) from [here](https://github.com/esnet/iperf/releases)

### Docker

[Docker](https://docs.docker.com/install/) and [Docker Compose](https://docs.docker.com/compose/install/).

### cloud-init

You can also install *trafic* automatically on cloud environments that support initialisation through cloud-init scripts. Refer to `cloud-init/trafic.ini`. This file provides you with an automated installation method.

Once the boot process is completely finalised, you will find a *trafic* installation for the default user (`ubuntu`)

*Further customisations*: You may need to customise the networking and user management (i.e. by providing your public keys for installation in the .ssh directory)

*Requirements*: Ubuntu bionic amd64 cloud image

*CAVEAT*: This installation will install stock iperf3. Although it should suffice, be aware that there is a [reported bug on iperf3 regarding the use of the parameter '-n' for TCP flows](https://github.com/esnet/iperf/issues/768), that is still open.

## Build and install

If you clones this repository directly, i.e. not using `go get`

```
go install ./...
```

---

[Trafic](https://en.wikipedia.org/wiki/Trafic) is not a typo, it's a film by [Jacques Tati](https://en.wikipedia.org/wiki/Jacques_Tati).

---

# trafic

At its core, *trafic* is just a flow scheduler.

You describe one or more flows, for example specifying which transport protocol (and possibly its congestion controller), transmission patterns, markings, etc. and let *trafic* run the client and server side of that flow at the specified time.  Each side of the flow is driven by a different *trafic* instance, sharing the same configuration as its peer.

When a flow completes, the *trafic* client stores key performance indicators (KPI) associated with each scheduled flow (e.g., bandwidth, packet loss, RTT, jitter).

These KPIs are sampled at a configurable rate (e.g. every 200ms) and made available as (per-flow) CSV files.  If requested, they can also be sent to an [Influx](https://www.influxdata.com/time-series-platform/influxdb/) instance where KPIs are organised as time-series and can be conveniently [queried](https://docs.influxdata.com/influxdb/v1.5/query_language/data_exploration/).

On top of that, *trafic* also provides the ability to define a traffic mix at high level, in terms of:
* How much bandwidth does the mix take;
* Which applications compose the mix (ABR video, web browsing, real-time A/V, etc.);
* Which percentage of bandwidth is used by each application.

## Modelling a flow

A flow is completely described by a YAML file.  The file has three parts:
* General;
* Client specific;
* Server specific.

The `client` and `server` sections are further subdivided into two parts: an `at` block containing scheduling information, and a role specific configuration.  (See the full list of [client](etc/client-blueprint.yaml) and [server](etc/server-blueprint.yaml) configurables.)

The direction of a flow is always server to client.

### Example: ABR video

To give an example, the following file describes an adaptive bit rate (ABR) video flow composed of seven HD (960x720) chunks, 10 seconds worth of video each:
```
label: &l abr-video

port: &p 5400

client:
  at:
    - 0s
    - 5s
    - 15s
    - 25s
    - 35s
    - 45s
    - 55s
  config:
    server-address: trafic-server.example.org.
    server-port: *p
    title: *l
    bytes: 1.8M
    report-interval-s: 0.2

server:
  at:
    - 0s
  config:
    server-port: *p
    report-interval-s: 0.2
```

The flow is given a label, "abr-video", which is used internally by the scheduler to reference the flow.  The label _must be unique_ among flows in the same mix.  Note that the flow is TCP unless otherwise specified.

The server side of the flow is instantiated once, at the very start of the scheduler execution (`0s`).  It will bind port 5400, and will do its KPI-sampling every 200ms.

A new client instance is scheduled to run every 10 seconds (at `5s`, `15s` and so on), simulating the typical refill pattern of the client's playout buffer.  The client downloads 1.8MB worth of data, that is (roughly) a 10s HD video chunk.  Note that the second chunk is fetched after 5s from the first, i.e., halfway through the playout of the first chunk..

Each client instance will connect to port `5400` on host `trafic-server.example.org`.

### Example: realtime audio

The following configuration models two instances of a mono-directional realtime audio flow (half of a typical Skype call), with regularly paced UDP packets bearing 126 bytes of RTP and media payload (`length`) aiming at constantly injecting 64Kbps (`target-bitrate`) in the network.  The two flows run, in parallel, for 60 seconds.

Things worth noting:
* An UDP flow needs to be explicitly declared using `udp: true`;
* A constant bitrate flow needs to be temporally bounded: `time-s: 60`;
* Parallelism of flows can be specified with the `parallel` keyword;

```
label: &l rt-audio

port: &p 5000

instances: &i 2

client:
  at:
    - 0s
  config:
    server-address: trafic-server.example.org.
    server-port: *p
    time-s: 60
    udp: true
    length: 126
    target-bitrate: 64K
    title: *l
    report-interval-s: 0.2
    parallel: *i

server:
  at:
    - 0s
  config:
    server-port: *p
    report-interval-s: 0.2
```

## Designing a traffic mix

A traffic mix is completely described by a YAML file.

The file has a header section defining the bandwidth to fill (`total-bandwidth`) and for how long the mix shall run (`total-time`).

```
# target aggregate bandwidth in bytes/sec (B/KB/MB/GB/TB)
total-bandwidth: 12.5MB

# how long the mix should run - expressed as duration (s, m, h, etc.)
total-time: 60s

# measure sampling tick
report-interval: 0.2s
```

The high level description of the application flows is done in the `flows` section.

Each flow defines its `kind`, i.e. the application it simulates.  The available pre-defined applications are:
* `realtime-audio` - one direction of a Skype / WebRTC voice call;
* `realtime-video` - one direction of a Skype / WebRTC video call;
* `scavenger` - (this is a bad name, I agree) an application-limited flow;
* `greedy` - a network-limited flow;
* `abr-video` - an (not so) adaptive bit rate video download
* `web-page` - average (~1.2MB) web page download

The amount of bandwidth this application consumes out of the total available (`total-bandwidth`) is given as a percentage using the `percent-bandwidth` keyword.

The ports used by the server are specified as a range using the `ports-range` keyword.

Application specific properties (_TODO document which_) are supplied in the `props` section.

```
server: &srv trafic-server.example.org.

flows:
  - kind: realtime-audio
    percent-bandwidth: 1%
    ports-range: 5000-5099
    props:
      label: rt-audio
      server: *srv
```

## Running the mix

Let's assume you have either manually or automatically (using `mixer`) synthesized your traffic mix, and successfully saved the relevant configuration files under one or more folders `DIR1..DIRn`.

You will then start server side:
```
s> schedule servers --flows-dirs=DIR1,...,DIRn --log-tag=TS --stats-dir=/tmp/trafic/servers-stats

[TS] 2018/06/08 15:01:22 common.go:309: 2018-06-08 16:01:22.328833 +0100 BST m=+0.103762416 -> deadline elapsed for abr-video
[TS] 2018/06/08 15:01:22 runner.go:41: Starting /usr/local/bin/iperf3 --server --json --interval 0.2 --port 5400
[TS] 2018/06/08 15:01:22 common.go:309: 2018-06-08 16:01:22.328833 +0100 BST m=+0.103762416 -> deadline elapsed for rt-audio
[TS] 2018/06/08 15:01:22 runner.go:41: Starting /usr/local/bin/iperf3 --server --json --interval 0.2 --port 5000
[TS] 2018/06/08 15:01:22 runner.go:52: Waiting for /usr/local/bin/iperf3 (PID=75505) to complete
[TS] 2018/06/08 15:01:22 runner.go:52: Waiting for /usr/local/bin/iperf3 (PID=75506) to complete
```

and subsequently start client side:
```
c> schedule clients --flows-dirs=DIR1,...,DIRn --log-tag=TC --stats-dir=/tmp/trafic/clients-stats

[TC] 2018/06/08 15:02:21 common.go:309: 2018-06-08 16:02:21.215852 +0100 BST m=+0.109531033 -> deadline elapsed for abr-video
[TC] 2018/06/08 15:02:21 runner.go:41: Starting /usr/local/bin/iperf3 --client trafic-server.example.org. --bytes 1.8M --get-server-output --reverse --title abr-video --json --interval 0.2 --port 5400
[TC] 2018/06/08 15:02:21 common.go:309: 2018-06-08 16:02:21.215852 +0100 BST m=+0.109531033 -> deadline elapsed for rt-audio
[TC] 2018/06/08 15:02:21 runner.go:41: Starting /usr/local/bin/iperf3 --client trafic-server.example.org. --length 126 --time 60 --get-server-output --parallel 2 --reverse --bitrate 64K --title rt-audio --udp --json --interval 0.2 --port 5000
[TC] 2018/06/08 15:02:21 runner.go:52: Waiting for /usr/local/bin/iperf3 (PID=75583) to complete
[TC] 2018/06/08 15:02:21 runner.go:52: Waiting for /usr/local/bin/iperf3 (PID=75584) to complete
2018/06/08 16:02:21 client abr-video finished ok
2018/06/08 16:02:21 1 client(s) to go

[...]

[TC] 2018/06/08 15:03:16 runner.go:41: Starting /usr/local/bin/iperf3 --client trafic-server.example.org. --bytes 1.8M --get-server-output --reverse --title abr-video --json --interval 0.2 --port 5400
[TC] 2018/06/08 15:03:16 runner.go:52: Waiting for /usr/local/bin/iperf3 (PID=75656) to complete
2018/06/08 16:03:16 client abr-video finished ok
2018/06/08 16:03:16 1 client(s) to go
2018/06/08 16:03:21 client rt-audio finished ok
2018/06/08 16:03:21 all currently active client(s) finished ok
```

When the client has successfully completed - _all currently active client(s) finished ok_ - you can safely kill the two sides of the scheduler.

The `/tmp/trafic/clients-stats` folder contains one JSON stats file for each flow that has been scheduled (i.e., one per `iperf3 -c` instance) and its companion CSV file that has been synthesised from the JSON source for simplify plotting, statistical analysis, etc.  For example, the realtime audio flow described above produces these two files:
```
20180608172323_client_rt-audio.csv
20180608172323_client_rt-audio.json
```
while the ABR video, which is made of seven independent flows (one per chunk), produces the following:
```
20180608172223_client_abr-video.csv
20180608172223_client_abr-video.json
20180608172228_client_abr-video.csv
20180608172228_client_abr-video.json
20180608172238_client_abr-video.csv
20180608172238_client_abr-video.json
20180608172248_client_abr-video.csv
20180608172248_client_abr-video.json
20180608172258_client_abr-video.csv
20180608172258_client_abr-video.json
20180608172308_client_abr-video.csv
20180608172308_client_abr-video.json
20180608172318_client_abr-video.csv
20180608172318_client_abr-video.json
```

### Using InfluxDB to store the stats

If you have a InfluxDB node at hand, you can use it to store the *trafic* stats as time series.  It suffices to run the client with the right `--influxdb-...` flags set.  For example:

```
   --influxdb-enabled \
   --influxdb-endpoint=http://influxdb:8086 \
   --influxdb-db=mydb \
   --influxdb-measurements=mymeasure-$(date +%s)
```

## Exploring KPIs

###  UDP
```
Timestamp,FlowID,FlowType,ToS,Bytes,BitsPerSecond,Jitterms,Packets,LostPackets,LostPercent
1528474943.000000,rt-audio_1528474943_6,udp,0x00,1638,65423.509657,0.033487,13,0,0.000000
1528474943.000000,rt-audio_1528474943_8,udp,0x00,1638,65418.603834,0.017496,13,0,0.000000
[...]
```

### TCP
```
Timestamp,FlowID,FlowType,ToS,PMTU,Bytes,BitsPerSecond,Retransmissions,SenderCWND,RTTms,RTTvar
1528476246.000000,abr-video_1528476246_5,tcp,0x00,0,2636004,7486213878.098011,0,0,0.000000,0.000000
```

### InfluxDB

TODO example queries

# flowsim

iperf3 is a good traffic generator, but it has its limitations. While developing `trafic`, an [issue](https://github.com/esnet/iperf/issues/768) regarding setting the total bytes transferred on a TCP stream was discovered. In order to accurately simulate web-short and ABR video streams, an additional simulator was developed. It follows the philosophy of iperf3 (server and client mode in one application).

*CAVEAT:* The integration of `flowsim` into `trafic` is still *work in progress*.

## flowsim modes

`flowsim` can be started as a TCP or QUIC server or client,  or as a UDP source or sink. It supports IPv4 and IPv6 addressing. By default, the server and sink modes use the IPv4 loopback address (`127.0.0.1`) by default. Interface addresses have to be set explicitly.


## flowsim as a TCP server

Once started as a server, `flowsim` will basically sit there and wait for the client to request bunches of data over a TCP connection.

```
Usage:
  flowsim server [flags]

Flags:
  -T, --TOS string   Value of the DSCP field in the IP layer (number or DSCP id) (default "CS0")
  -h, --help         help for server
  -I, --ip string    IP address or host name bound to the flowsim server (default "127.0.0.1")
  -1, --one-off      Just accept one connection and quit (default is run until killed)
  -p, --port int     TCP port bound to the flowsim server (default 8081)
  -Q, --quic         Use QUIC (default is TCP)
```

Note in the normal mode, `flowsim` will be executed until killed with a `SIGINT` sinal (i.e. `Control-C` from the keyboard). The `--one-off` option will make `flowsim` quit after a flow has been served.

The size of the TCP PDU served and the moment where a connection is closed are determined by the client.

## flowsim as a TCP client

When `flowsim` is started as a client, a number of TCP segments with a fixed size will be requested from the server. All segments will be served over the same TCP connection, which is closed afterwards.

```
Usage:
  flowsim client [flags]

Flags:
  -T, --TOS string     Value of the DSCP field in the IP packets (valid int or DSCP-Id) (default "CS0")
  -N, --burst string   Size of each burst (as x(.xxx)?[kmgtKMGT]?) (default "1M")
  -h, --help           help for client
  -t, --interval int   Interval in secs between bursts (default 10)
  -I, --ip string      IP address or host name of the flowsim server to talk to (default "127.0.0.1")
  -n, --iter int       Number of bursts (default 6)
  -p, --port int       TCP port of the flowsim server (default 8081)
  -Q, --quic           Use QUIC (default is TCP)
```

## flowsim as a UDP source

```
Usage:
  flowsim source [flags]

Flags:  -T, --TOS string      Value of the DSCP field in the IP packets (valid int or DSCP-Id) (default "CS0")
  -h, --help            help for source
  -I, --ip string       IP address or host name of the flowsim UDP sink to talk to (default "127.0.0.1")
  -L, --local string    Outgoing source IP address to use; determins interface (default: empyt-any interface)
  -N, --packet string   Size of each packet (as x(.xxx)?[kmgtKMGT]?) (default "1k")
  -p, --port int        UDP port of the flowsim UDP sink (default 8081)
  -P, --pps int         Packets per second (default 10)
  -t, --time int        Total time sending (default 6)
  -v, --verbose         Print info re. all generated packets```

## flowsim as a UDP sink

```
Usage:
  flowsim sink [flags]

Flags:
  -h, --help        help for sink
  -I, --ip string   IP address or host name to listen on for the flowsim UDP sink (default "127.0.0.1")
  -m, --multi       Stay in the sink forever and print stats for multiple incoming streams
  -p, --port int    UDP port of the flowsim UDP sink (default 8081)
  -v, --verbose     Print per packet info
```
