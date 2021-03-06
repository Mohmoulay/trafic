package mixer

import "path"

var defaultRealtimeVideoTmpl string = `
{{/*
  Models one direction of a real-time video stream
  810Kbps, 930 bytes of RTP + media payload

  Configuration keys:
  - label: a label added to the final report
  - server: name or address of the server side of the flow
  - port: the UDP port to use when connecting to the server
  - instances: how many instances of the flow to run in parallel
  - time: how long the flow needs to live for
  - report_interval: flow measures sampling timer
*/}}

label: &l {{ .Label }}

port: &p {{ .Port }}

instances: &i {{ .Instances }}

client:
  at:
    - 0s
  config:
    server-address: {{ .Server }}
    server-port: *p
    time-s: {{ .Time }}
    udp: true
    length: 930
    target-bitrate: 810K
    title: *l
    report-interval-s: {{ .ReportInterval }}
    parallel: *i

server:
  at:
    - 0s
  config:
    server-port: *p
    report-interval-s: {{ .ReportInterval }}
`

type RealtimeVideo struct{}

func NewRealtimeVideo() Mixer {
	return &RealtimeVideo{}
}

func (RealtimeVideo) WriteConf(baseDir string, g GlobalDesc, c FlowDesc) error {
	// target-bitrate: 810K
	return writeFixedBitrate(
		path.Join(baseDir, "realtime-video.yaml"),
		defaultRealtimeVideoTmpl,
		g, c, 810000,
	)
}

func (RealtimeVideo) Name() string {
	return "realtime-video"
}
