package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/xly0713/flv_parse/flv"
)

var (
	flvPath         string
	printVideoFrame bool
)

func init() {
	flag.StringVar(&flvPath, "f", "", "flv path")
	flag.BoolVar(&printVideoFrame, "pf", false, "print video frame info one by one")
}

func main() {
	flag.Parse()

	if flvPath == "" || !isFileExist(flvPath) {
		fmt.Println("valid flv path required")
		return
	}

	r, err := os.Open(flvPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer r.Close()

	fp := flv.NewFlvParser(r, printVideoFrame)
	if err := fp.ParseFlv(); err != nil {
		fmt.Printf("failed to parse flv, err: %v", err)
		return
	}

	flvHeader := fp.Header()
	fmt.Printf("flv header: %+v\n", flvHeader)

	//flvScriptInfo := fp.MetaInfo()
	//fmt.Printf("flv script data: %+v\n", flvScriptInfo)

	flvBodyInfo := fp.BodyInfo()
	fmt.Printf("flv body info: %+v\n", flvBodyInfo)

	fp.PrintMetaInfo()
}

func isFileExist(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil || os.IsExist(err)
}
