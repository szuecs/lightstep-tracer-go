package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"

	lightstep "github.com/lightstep/lightstep-tracer-go"
	opentracing "github.com/opentracing/opentracing-go"
)

type Carriers struct {
	TextMap map[string]string `json:"text_map"`

	Binary string `json:"binary"`
}

func main() {
	tracer := lightstep.NewTracer(lightstep.Options{
		AccessToken: "invalid",
	})

	var carriers Carriers
	if err := json.NewDecoder(os.Stdin).Decode(&carriers); err != nil {
		log.Println(carriers)
		fatal("could not read carriers from stdin: ", err)
	}

	spanContextTextMap, err := tracer.Extract(opentracing.TextMap, opentracing.TextMapCarrier(carriers.TextMap))
	if err != nil {
		fatal("could not read extract text map context: ", err)
	}

	b, err := base64.StdEncoding.DecodeString(carriers.Binary)
	if err != nil {
		fatal("could not base64 binary carrier: ", err)
	}

	spanContextBinary, err := tracer.Extract(opentracing.Binary, bytes.NewBuffer(b))
	if err != nil {
		fatal("could not extract extract binary context: ", err)
	}

	binOut := bytes.NewBuffer(nil)
	if err := tracer.Inject(spanContextBinary, opentracing.Binary, binOut); err != nil {
		fatal("could not inject binary context: ", err)
	}

	output := Carriers{
		TextMap: make(map[string]string),
		Binary:  base64.StdEncoding.EncodeToString(binOut.Bytes()),
	}

	err = tracer.Inject(spanContextTextMap, opentracing.TextMap, opentracing.TextMapCarrier(output.TextMap))
	if err != nil {
		fatal("could not inject text map context: ", err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
		fatal("could not marshal json to stdout: ", err)
	}
	os.Exit(0)
}

func fatal(args ...interface{}) {
	fmt.Println(args...)
	os.Exit(1)
}
