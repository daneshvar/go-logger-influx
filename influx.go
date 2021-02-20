package logger

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	influxdb "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

const measurement = "logs"

type Influx struct {
	pool     sync.Pool
	client   influxdb.Client
	writeAPI api.WriteAPI
	app      string
}

func InfluxWriter(serverURL string, authToken string, org string, bucket string, app string, caller bool, stack EnablerFunc, enabler EnablerFunc) *Writer {
	i := &Influx{
		pool: sync.Pool{New: func() interface{} {
			b := bytes.NewBuffer(make([]byte, 150)) // buffer init with 150 size
			b.Reset()
			return b
		}},
		app: app,
	}
	i.Connect(serverURL, authToken, org, bucket)
	return newWriter(enabler, stack, caller, i)
}

func (i *Influx) Connect(serverURL string, authToken string, org string, bucket string) {
	i.client = influxdb.NewClient(serverURL, authToken)
	i.writeAPI = i.client.WriteAPI(org, bucket) // https://docs.influxdata.com/influxdb/v2.0/write-data/
}

func (i *Influx) close() {
	// Force all unwritten data to be sent
	i.writeAPI.Flush()
	// Ensures background processes finishes
	i.client.Close()
}

func (i *Influx) getBuffer() *bytes.Buffer {
	return i.pool.Get().(*bytes.Buffer)
}

func (i *Influx) putBuffer(b *bytes.Buffer) {
	b.Reset()
	i.pool.Put(b)
}

func (i *Influx) Print(l Level, s string, caller string, stack []string, message string) {
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
			"level": levelText[l],
		},
		fields,
		time.Now())

	// write asynchronously
	i.writeAPI.WritePoint(p)
}

func (i *Influx) Printv(l Level, s string, caller string, stack []string, message string, keysValues []interface{}) {
	fields := make(map[string]interface{})

	fields["message"] = message
	if caller != "" {
		fields["caller"] = caller
	}

	if len(stack) > 0 {
		fields["stack"] = strings.Join(stack, "\r\n")
	}

	i.addKeyValues(fields, keysValues)

	// create point
	p := influxdb.NewPoint(
		measurement,
		map[string]string{
			"app":   i.app,
			"scope": s,
			"level": levelText[l],
		},
		fields,
		time.Now())

	// write asynchronously
	i.writeAPI.WritePoint(p)
}

func (i *Influx) addKeyValues(fields map[string]interface{}, keysValues []interface{}) {
	lenValues := len(keysValues)
	if lenValues > 1 {
		values := i.getBuffer()
		defer i.putBuffer(values)

		// first key=value
		values.WriteString(fmt.Sprintf("%v=%v", keysValues[0], keysValues[1]))
		for i := 2; i < lenValues; i += 2 {
			values.WriteString(fmt.Sprintf("\r\n%v=%v", keysValues[i], keysValues[i+1]))
		}

		fields["values"] = values.Bytes()
	}
}
