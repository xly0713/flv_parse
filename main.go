package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/xly0713/flv_parse/flv"
)

func main() {
	fs := flag.NewFlagSet("flv-parse", flag.ExitOnError)
	var (
		flvPath = fs.String("f", "", "The flv video file path")
		verbose = fs.Bool("verbose", false, "Show every frame info of the flv video file")
	)
	fs.Usage = usageFor(fs, os.Args[0]+" [flags]")
	_ = fs.Parse(os.Args[1:])

	if *flvPath == "" || !isFileExist(*flvPath) {
		fmt.Println("flv video file path required")
		return
	}

	r, err := os.Open(*flvPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer r.Close()

	fp := flv.NewFlvParser(r, *verbose)
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

func usageFor(fs *flag.FlagSet, short string) func() {
	return func() {
		fmt.Fprintf(os.Stderr, "USAGE\n")
		fmt.Fprintf(os.Stderr, " %s\n", short)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "FLAGS\n")
		w := tabwriter.NewWriter(os.Stderr, 0, 2, 2, ' ', 0)
		fs.VisitAll(func(f *flag.Flag) {
			fmt.Fprintf(w, "\t-%s \t%s (default: %s)\n", f.Name, f.Usage, f.DefValue)
		})
		w.Flush()
		fmt.Fprintf(os.Stderr, "\n")
	}
}

func isFileExist(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil || os.IsExist(err)
}
