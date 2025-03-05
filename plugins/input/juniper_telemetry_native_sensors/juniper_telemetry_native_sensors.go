//go:generate ../../../tools/config_includer/generator
//go:generate ../../../tools/readme_config_includer/generator
package juniper_telemetry_native_sensors

import (
	_ "embed"
	"net"
	"sync"
	"time"
	"fmt"
	"github.com/influxdata/telegraf"
	// "github.com/influxdata/telegraf/internal"
	"github.com/influxdata/telegraf/plugins/common/socket"
	"github.com/influxdata/telegraf/plugins/inputs"
)

//go:embed sample.conf
var sampleConfig string

var once sync.Once

type juniper_telemetry_native_sensors struct {
	ServiceAddress string          `toml:"service_address"`
	TimeSource     string          `toml:"time_source"`
	Log            telegraf.Logger `toml:"-"`
	socket.Config
	socket.SplitConfig

	socket *socket.Socket
	parser telegraf.Parser
}

func (*juniper_telemetry_native_sensors) SampleConfig() string {
	return sampleConfig
}

func (sl *juniper_telemetry_native_sensors) SetParser(parser telegraf.Parser) {
	sl.parser = parser
}

func (sl *juniper_telemetry_native_sensors) Init() error {
    if sl.ServiceAddress == "" {
        sl.Log.Error("ServiceAddress is empty")
        return fmt.Errorf("ServiceAddress is empty")
    }

    sl.Log.Infof("Initializing socket with ServiceAddress: %s", sl.ServiceAddress)
	sock, err := sl.Config.NewSocket(sl.ServiceAddress, &sl.SplitConfig, sl.Log)
	if err != nil {
		return err
	}
	sl.socket = sock

	return nil
}

func (sl *juniper_telemetry_native_sensors) Start(acc telegraf.Accumulator) error {
	// Create the callbacks for parsing the data and recording issues
	onData := func(_ net.Addr, data []byte, receiveTime time.Time) {
		metrics, err := sl.parser.Parse(data)

		if err != nil {
			acc.AddError(err)
			return
		}

		if len(metrics) == 0 {
			once.Do(func() {
				// sl.Log.Debug(internal.NoMetricsCreatedMsg)
				sl.Log.Debug("internal.NoMetricsCreatedMsg")
			})
		}

		for _, m := range metrics {
			switch sl.TimeSource {
			case "", "metric":
			case "receive_time":
				m.SetTime(receiveTime)
			}

			acc.AddMetric(m)
		}
	}
	onError := func(err error) {
		acc.AddError(err)
	}

	// Start the listener
	if err := sl.socket.Setup(); err != nil {
		return err
	}
	sl.socket.Listen(onData, onError)
	addr := sl.socket.Address()
	sl.Log.Infof("Listening on %s://%s", addr.Network(), addr.String())

	return nil
}

func (*juniper_telemetry_native_sensors) Gather(telegraf.Accumulator) error {
	return nil
}

func (sl *juniper_telemetry_native_sensors) Stop() {
	if sl.socket != nil {
		sl.socket.Close()
	}
}

func init() {
	inputs.Add("juniper_telemetry_native_sensors", func() telegraf.Input {
		return &juniper_telemetry_native_sensors{}
	})
}
