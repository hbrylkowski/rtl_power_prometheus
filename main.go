package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var (
	bins = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "frequency_power",
		Help: "Power at given frequency",
	},
		[]string{"frequency"},
	)
)

func main() {
	lower := flag.String("lower_frequency", "", "Lower frequency limit")
	upper := flag.String("upper_frequency", "", "Upper frequency limit")
	bin := flag.String("bin_size", "", "Size of each frequency bin")

	flag.Parse()

	if *lower == "" || *upper == "" || *bin == "" {
		fmt.Println("All flags 'lower_frequency', 'upper_frequency', and 'bin_size' are required.")
		os.Exit(1)
	}
	frequency := *lower + ":" + *upper + ":" + *bin

	cmd := exec.Command("rtl_power", "-f", frequency, "-")

	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(cmdReader)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			data := strings.Split(line, ", ")

			freq, err := strconv.ParseFloat(data[2], 64)
			if err != nil {
				panic(err)
			}

			step, err := strconv.ParseFloat(data[4], 64)
			if err != nil {
				panic(err)
			}
			for _, power := range data[6:] {
				powerFloat, err := strconv.ParseFloat(power, 64)
				if err != nil {
					panic(err)
				}
				bins.WithLabelValues(fmt.Sprintf("%d", int64(freq))).Set(powerFloat)
				freq = freq + step
			}
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		err = http.ListenAndServe(":2112", nil)
		if err != nil {
			panic(err)
		}
	}()
	err = cmd.Start()
	if err != nil {
		panic(err)
	}
	err = cmd.Wait()
	if err != nil {
		panic(err)
	}
}
