package logger

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/daneshvar/go-logger"
	influxdb "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

const measurement = "logs"

type Influx struct {
	// pool     sync.Pool
	client   influxdb.Client
	writeAPI api.WriteAPI
	app      string
}

func InfluxWriter(serverURL string, authToken string, org string, bucket string, app string, caller bool, stack logger.EnablerFunc, enabler logger.EnablerFunc) *logger.Writer {
	encoder := &Influx{
		// pool: sync.Pool{New: func() interface{} {
		// 	b := bytes.NewBuffer(make([]byte, 150)) // buffer init with 150 size
		// 	b.Reset()
		// 	return b
		// }},
		app: app,
	}
	encoder.Connect(serverURL, authToken, org, bucket)

	return logger.NewWriter(enabler, stack, caller, encoder)
}

func (i *Influx) Connect(serverURL string, authToken string, org string, bucket string) {
	i.client = influxdb.NewClient(serverURL, authToken)
	i.writeAPI = i.client.WriteAPI(org, bucket) // https://docs.influxdata.com/influxdb/v2.0/write-data/
}

func (i *Influx) Close() {
	// Force all unwritten data to be sent
	i.writeAPI.Flush()
	// Ensures background processes finishes
	i.client.Close()
}

// func (i *Influx) getBuffer() *bytes.Buffer {
// 	return i.pool.Get().(*bytes.Buffer)
// }

// func (i *Influx) putBuffer(b *bytes.Buffer) {
// 	b.Reset()
// 	i.pool.Put(b)
// }

func (i *Influx) Print(l logger.Level, s string, caller string, stack []string, message []interface{}) {
	fields := make(map[string]interface{})

	fields["message"] = fmt.Sprint(message...)
	if caller != "" {
		fields["caller"] = caller
	}

	if len(stack) > 0 {
		fields["stack"] = strings.Join(stack, "\r\n")
	}

	// create point
	p := influxdb.NewPoint(
		measurement,
		map[string]string{
			"app":   i.app,
			"scope": s,
			"level": logger.LevelText(l),
		},
		fields,
		time.Now())

	// write asynchronously
	i.writeAPI.WritePoint(p)
}

func (i *Influx) Printv(l logger.Level, s string, caller string, stack []string, message string, keysValues []interface{}) {
	fields := make(map[string]interface{})

	fields["message"] = message
	if caller != "" {
		fields["caller"] = caller
	}

	if len(stack) > 0 {
		fields["stack"] = strings.Join(stack, "\r\n")
	}

	i.addKeyValues(fields, keysValues)

	jsonString, _ := json.Marshal(fields)

	// create point
	p := influxdb.NewPoint(
		measurement,
		map[string]string{
			"app":   i.app,
			"scope": s,
			"level": logger.LevelText(l),
		},
		map[string]interface{}{
			"values": jsonString,
		},
		time.Now())

	// write asynchronously
	i.writeAPI.WritePoint(p)
}

func (i *Influx) Prints(l logger.Level, s string, caller string, stack []string, message string) {
	fields := make(map[string]interface{})

	fields["message"] = message
	if caller != "" {
		fields["caller"] = caller
	}

	if len(stack) > 0 {
		fields["stack"] = strings.Join(stack, "\r\n")
	}

	// create point
	p := influxdb.NewPoint(
		measurement,
		map[string]string{
			"app":   i.app,
			"scope": s,
			"level": logger.LevelText(l),
		},
		fields,
		time.Now())

	// write asynchronously
	i.writeAPI.WritePoint(p)
}

func (i *Influx) Printf(l logger.Level, s string, caller string, stack []string, format string, args []interface{}) {
	i.Prints(l, s, caller, stack, fmt.Sprintf(format, args...))
}

func (i *Influx) addKeyValues(fields map[string]interface{}, keysValues []interface{}) {
	lenValues := len(keysValues)
	if lenValues < 1 {
		return
	}

	for i := 0; i < lenValues; i += 2 {
		if i+1 < lenValues {
			fields[fmt.Sprintf("%v", keysValues[i])] = keysValues[i+1]
		} else {
			fields[fmt.Sprintf("%v", keysValues[i])] = "!VALUE"
		}
	}
}
