package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var initZone string
var timeFormat = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
var ISO8601 = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} [+-]\d{4}$`)
var findISO8601 = regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} [+-]\d{4}`)
var RFC3339 = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}$`)

func ISO8601_to_RFC3339(t string) (string, error) {
	if !ISO8601.MatchString(t) {
		return t, errors.New("time string is not ISO8601 format.")
	}
	return fmt.Sprintf("%sT%s%s:%s", t[0:10], t[11:19], t[20:23], t[23:25]), nil
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// how to get local timezone offset value
	_, offset := time.Now().Zone()
	if offset > 0 {
		initZone = fmt.Sprintf("+%04d", offset/60/60*100)
	} else {
		initZone = fmt.Sprintf("-%04d", (-1*offset)/60/60*100)
	}
}

func main() {
	startPtr := flag.String("start", "2018-01-01", "start date")
	endPtr := flag.String("end", "2019-12-31", "end date")
	zonePtr := flag.String("zone", initZone, "zone offset time")
	debugPtr := flag.Bool("debug", false, "debug mode")
	helpPtr := flag.Bool("help", false, "print help")
	flag.Parse()
	if *helpPtr {
		flag.PrintDefaults()
		os.Exit(0)
	}
	if !timeFormat.MatchString(*startPtr) {
		fmt.Println("not matching start date format. must be 0000-00-00")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if !timeFormat.MatchString(*endPtr) {
		fmt.Println("not matching end date format. must be 0000-00-00")
		flag.PrintDefaults()
		os.Exit(1)
	}
	cmd := exec.Command(
		"git",
		"--no-pager",
		"log",
		"--reverse",
		"--date=iso",
		`--pretty=format:%ad %an %s`,
		fmt.Sprintf(`--after="%s 00:00:00 %s"`, *startPtr, *zonePtr),
		fmt.Sprintf(`--before="%s 23:59:59 %s"`, *endPtr, *zonePtr),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if stderr.String() != "" {
		fmt.Fprintf(os.Stderr, stderr.String())
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, stderr.String())
		os.Exit(1)
	}
	total, err := time.ParseDuration("0h")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if stdout.String() == "" {
		fmt.Printf("%s~%s : %s\n", *startPtr, *endPtr, total)
		os.Exit(0)
	}

	var before time.Time
	for n, l := range strings.Split(stdout.String(), "\n") {
		getTime := findISO8601.FindString(l)
		if getTime == "" {
			continue
		}
		rfctime, err := ISO8601_to_RFC3339(getTime)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		t, err := time.Parse(time.RFC3339, rfctime)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		elapsed := t.Sub(before)
		if *debugPtr {
			if n != 0 {
				fmt.Println(elapsed, ">")
			}
			fmt.Println("\t", l)
		}
		h, err := time.ParseDuration("1h")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		if elapsed < h*2 {
			total += elapsed
		} else {
			total += h
		}
		before = t
	}
	fmt.Printf("%s~%s : %s\n", *startPtr, *endPtr, total)
}
