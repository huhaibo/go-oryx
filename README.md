# go-oryx

<a href="https://godoc.org/github.com/ossrs/go-oryx">
    <img src="https://godoc.org/github.com/ossrs/go-oryx?status.svg" alt="GoDoc">
</a>
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/ossrs/go-oryx?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

The go-oryx is [SRS++][srs], focus on real-time live streaming cluster.

## Usage

For linux/unix-like os:

```
go get github.com/ossrs/go-oryx &&
cd $GOPATH/src/github.com/ossrs/go-oryx &&
$GOPATH/bin/go-oryx -c conf/oryx.json
```

Or, for windows:

```
go get github.com/ossrs/go-oryx &&
cd %GOPATH%\src\github.com\ossrs\go-oryx &&
%GOPATH%\bin\go-oryx.exe -c conf\oryx.json
```

About how to build and run at current directory:

```
cd $GOPATH/src/github.com/ossrs/go-oryx &&
go build . && ./go-oryx -c conf/oryx.json
```

About how to set $GOPATH, read [prepare go][go-prepare].

## IDE

GO SDK: [download][go-download]

JetBrains IntelliJ IDEA: [download][go-ide]

IntelliJ IDEA Golang Plugin: [repository][go-ide-plugin], [download][go-ide-plugin-download]

## SRS vs go-oryx

Why rewrite the SRS to go-oryx:

1. Coroutine Or Goroutine: SRS base on ST, it’s more important than c/c++ language for streaming server; while golang support goroutine, which is actually what ST do.
1. Multiple Processes: SRS is single process. It’s too weak for modern server with 16 or 64 CPUs and 2 or 4 10Gbps network. Multiple cpus and network interfaces requires multiple processes server.
1. New Arch: New arch for HTTP/RTMP/FLV/HLS/RTSP or other protocol, it’s better to support many protocol especially private protocol, which will provides realtime streaming.

### Features

1. v0.1.0 Supports Multiple Processes.
1. v0.1.0 Supports Linux, Unix-like and Windows.
1. v0.1.1 Supports JSON style config file.
1. v0.1.2 [#41](../../issues/41) Supports Reload config file.
1. v0.1.3 Standard godoc, gofmt, gotest and TravisCI.
1. v0.1.4 Support daemon over [ossrs/go-daemon][go-daemon](fork from [sevlyar/go-daemon][fork-go-daemon]).
1. v0.1.5 Extend JSON with c++ style comments.
1. v0.1.6 Support heartbeat to report for ARM.
1. v0.1.7 Use agent(source+channel+sink) to build complex stream river.
1. v0.1.8 [#37](../../issues/37) Supports Publish and Play VP6 RTMP stream.
1. v0.1.9 Supports Delivery VP6/H.264 and Speex/AAC/MP3/Nellymoser codec.
1. v0.1.10 Supports 10k(8CPUs) for RTMP players.
1. v0.1.11 Supports 10k(4CPUs) for RTMP players.
1. v0.1.12 Supports 10k(3CPUs) for RTMP players.
1. v0.1.13 Supports 10k(2CPUs) for RTMP players.
1. v0.1.14 Supports SRS style config file.
1. v0.1.15 Supports LOG+, the connection-based tracable log.
1. v0.1.16 Supports debug rtmp recv by config debug.rtmp_dump_recv.
1. v0.1.17 Supports force to realtime mode by config vhost.min_latency.
1. [dev] Supports gop-cache and drop frame strategy.
1. [plan] [#45](../../issues/45) HLS: Support audio only HLS stream.
1. [plan] [#44](../../issues/44) DASH: Support remux stream to MPEG-DASH
1. [plan] [#43](../../issues/43) RTSP: Support push RTSP to server.
1. [plan] [#42](../../issues/42) UDP: Support push MPEG-TS over UDP to server.
1. [plan] [#40](../../issues/40) HDS: Support remux stream to HDS.
1. [plan] [#39](../../issues/39) API: Support HTTP callbacks
1. [plan] [#38](../../issues/38) API: Support HTTP API for client to access.
1. [plan] [#36](../../issues/36) FLV: Remux stream to HTTP-FLV streaming.
1. [plan] [#35](../../issues/35) HLS: Support HLS+ Edge Cluster.
1. [plan] [#24](../../issues/24) RTMP: Support standard uri like http.
1. [plan] [#46](../../issues/46) Codec: Support encode the audio and video device to stream.
1. [plan] [#47](../../issues/47) System: Support copy stream to another vhost.

Winlin 2015.10

[srs]: https://github.com/ossrs/srs

[go-download]: http://www.golangtc.com/download
[go-prepare]: http://blog.csdn.net/win_lin/article/details/40618671
[go-ide]: http://www.jetbrains.com/idea/download
[go-ide-plugin]: https://github.com/go-lang-plugin-org/go-lang-idea-plugin
[go-ide-plugin-download]: https://plugins.jetbrains.com/plugin/5047
[go-daemon]: http://github.com/ossrs/go-daemon
[fork-go-daemon]: http://github.com/sevlyar/go-daemon
