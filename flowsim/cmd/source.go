package cmd

import (
	"fmt"
	common "github.com/mami-project/trafic/flowsim/common"
	"github.com/mami-project/trafic/flowsim/udp"
	"github.com/spf13/cobra"
)

var sourceIp string
var sourcePort int
var sourceLocal string
var sourcePps int
var sourceTime int
var sourceTos string
var sourcePacket string
var sourceVerbose bool
var sourceIpv6 bool

// sourceCmd represents the source command
var sourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Start a flowsim UDP source",
	Long: `Will run flowsim as a UDP CBR source
and try to talk to a flowsim UPD sink.`,
	Run: func(cmd *cobra.Command, args []string) {
		pktsize, err := utoi(sourcePacket)
		if err != nil {
			fmt.Printf("Warning: %v, generating %d byte packets\n", err, pktsize)
		}

		tos, err := Dscp(sourceTos)
		if err != nil {
			fmt.Printf("Error decoding DSCP (%s): %v\n", sourceTos, err)
		}
		useIp, err := common.FirstIP(sourceIp, sourceIpv6)
		common.FatalError(err)

		udp.Source(useIp, sourcePort, sourceLocal,
			sourceTime, sourcePps,
			pktsize, tos*4, sourceVerbose)

	},
}

func init() {
	rootCmd.AddCommand(sourceCmd)

	sourceCmd.PersistentFlags().StringVarP(&sourceIp, "ip", "I", "localhost", "IP address or host name of the flowsim UDP sink to talk to")
	sourceCmd.PersistentFlags().IntVarP(&sourcePort, "port", "p", 8081, "UDP port of the flowsim UDP sink")
	sourceCmd.PersistentFlags().StringVarP(&sourceLocal, "local", "L", "", "Outgoing source IP address to use; determins interface (default: empyt-any interface)")
	sourceCmd.PersistentFlags().IntVarP(&sourceTime, "time", "t", 6, "Total time sending")
	sourceCmd.PersistentFlags().IntVarP(&sourcePps, "pps", "P", 10, "Packets per second")
	sourceCmd.PersistentFlags().StringVarP(&sourcePacket, "packet", "N", "1k", "Size of each packet (as x(.xxx)?[kmgtKMGT]?)")
	sourceCmd.PersistentFlags().StringVarP(&sourceTos, "TOS", "T", "CS0", "Value of the DSCP field in the IP packets (valid int or DSCP-Id)")
	sourceCmd.PersistentFlags().BoolVarP(&sourceVerbose, "verbose", "v", false, "Print info re. all generated packets")
	sourceCmd.PersistentFlags().BoolVarP(&sourceIpv6, "ipv6", "6", false, "Use IPv6 (default is IPv4)")
}
