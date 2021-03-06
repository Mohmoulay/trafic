package cruncher

import (
	"bytes"
	"fmt"
)

type InfluxDBCruncher struct {
	measurement string
}

func NewInfluxDBCruncher(measurement string) Cruncher {
	return &InfluxDBCruncher{
		measurement: measurement,
	}
}

// See https://docs.influxdata.com/influxdb/v1.5/write_protocols/line_protocol_reference/
//
// Each line, separated by the newline character \n, represents a single point
// in InfluxDB. Line Protocol is whitespace sensitive.
//
// A line-protocol line needs to be formatted like this:
// <measurement>[,<tag_key>=<tag_value>[,<tag_key>=<tag_value>]] <field_key>=<field_value>[,<field_key>=<field_value>] [<timestamp>]

func (c *InfluxDBCruncher) CrunchTCP(tcpFlowStats TCPFlowStats) ([]byte, error) {
	var lines bytes.Buffer

	start := tcpFlowStats.ServerOutputJSON.Start.Timestamp.Timesecs

	for _, interval := range tcpFlowStats.ServerOutputJSON.Intervals {
		for _, stream := range interval.Streams {
			flowID := formatFlowID(tcpFlowStats.Title, tcpFlowStats.Start.Cookie, start, stream.Socket)

			line := fmt.Sprintf("%s,flowid=%s,type=tcp,tos=0x%02x,pmtu=%d bytes=%d,bps=%f,rtx=%d,sndcwnd=%d,rtt_ms=%f,rtt_var=%f %d\n",
				c.measurement,
				flowID,
				tcpFlowStats.Start.TestStart.Tos,
				stream.Pmtu,
				stream.Bytes,
				stream.BitsPerSecond,
				stream.Retransmits,
				stream.SndCwnd,
				float64(stream.Rtt)/1000,
				float64(stream.Rttvar)/1000,
				makeInfluxDBTimestamp(start, stream.Start))

			// Don't bother about the bytes written, just check the return status
			_, err := lines.WriteString(line)
			if err != nil {
				return nil, err
			}
		}
	}

	return lines.Bytes(), nil
}

func (c *InfluxDBCruncher) CrunchUDP(udpFlowStats UDPFlowStats) ([]byte, error) {
	var lines bytes.Buffer

	start := udpFlowStats.Start.Timestamp.Timesecs

	for _, interval := range udpFlowStats.Intervals {
		for _, stream := range interval.Streams {
			flowID := formatFlowID(udpFlowStats.Title, udpFlowStats.Start.Cookie, start, stream.Socket)

			line := fmt.Sprintf("%s,flowid=%s,type=udp,tos=0x%02x bytes=%d,bps=%f,jitter=%f,pkts=%d,lost_pkts=%d,lost_percent=%f %d\n",
				c.measurement,
				flowID,
				udpFlowStats.Start.TestStart.Tos,
				stream.Bytes,
				stream.BitsPerSecond,
				stream.JitterMs,
				stream.Packets,
				stream.LostPackets,
				stream.LostPercent,
				makeInfluxDBTimestamp(start, stream.Start))

			_, err := lines.WriteString(line)
			if err != nil {
				return nil, err
			}
		}
	}

	return lines.Bytes(), nil
}

// InfluxDB timestamps are unix time in nanoseconds
func makeInfluxDBTimestamp(flowStart int, sampleStart float64) int {
	return int((float64(flowStart) + sampleStart) * 1e9)
}
