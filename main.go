package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/v29/github"
	"golang.org/x/oauth2"
)

func searchProtoOpts(ctx context.Context, page, perpage int, textmatch bool) (*github.CodeSearchResult, error) {

	// https://github.com/search?l=&p=4&q=in%3Afile+%22extend+google.protobuf%22+extension%3Aproto&ref=advsearch&type=Code&utf8=%E2%9C%93
	// l=
	// p=4
	// q
	// in:file
	// "extend+google.protobuf"
	// extension:proto
	//
	// ref=advsearch
	// type=Code
	// utf8=%E2%9C%93
	resul, _, err := client.Search.Code(ctx, `in:file 
	"extend google.protobuf" extension:proto 
	-filename:13_protobufoptions.proto
	-filename:bar.proto
	-filename:custom-options.proto
	-filename:custom_options.proto
	-filename:CustomOptions.proto
	-filename:data.proto
	-filename:desc_test_options.proto
	-filename:google_unittest_custom_options.proto
	-filename:helloworld.proto
	-filename:json.proto
	-filename:MyOptions.proto
	-filename:page.proto
	-filename:pb_test_04.option.proto
	-filename:php.proto
	-filename:protogen.proto
	-filename:steammessages_unified_base.steamclient.proto
	-filename:steammessages_unified_base.steamworkssdk.proto
	-filename:test.proto
	-filename:unittest_custom_options.proto
	-filename:unittest_custom_options_proto3.proto
	-filename:yara.proto	
`, &github.SearchOptions{
		TextMatch: textmatch,
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: perpage,
		},
	})
	return resul, err
}

var client *github.Client
var startPage = flag.Int("start_page", 0, "starting page")

func openf(name string, flag int) (*bufio.Writer, func()) {
	f1, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|flag, os.ModePerm)
	if err != nil {
		fmt.Printf("1 error %v\n", err)
		os.Exit(1)
	}
	buf := bufio.NewWriter(f1)
	f := func() {
		buf.Flush()
		f1.Close()
	}
	return buf, f
}

func main() {
	flag.Parse()
	ctx := context.Background()
	token := os.Getenv("GITHUB_AUTH_TOKEN")
	if token == "" {
		log.Fatal("Unauthorized: No token present")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client = github.NewClient(tc)

	shaMap := map[string]int{}
	shas, err := ioutil.ReadFile("sha_count.txt")
	if err == nil {
		lines := strings.Split(string(shas), "\n")
		for _, l := range lines {
			fields := strings.Split(l, "\t")
			if len(fields) != 2 {
				continue
			}
			c, err := strconv.ParseInt(fields[1], 10, 32)
			if err != nil {
				fmt.Printf("parse sha error %v\n", err)
				return
			}
			shaMap[fields[0]] = int(c)
		}
		for k, v := range shaMap {
			fmt.Printf("%s\t%d\n", k, v)
		}
	}

	tmf, f := openf("textmatch.txt", os.O_APPEND)
	defer f()
	resf, f := openf("results.txt", os.O_APPEND)
	defer f()
	shaCount, f := openf("sha_count.txt", os.O_TRUNC)
	defer f()
	pagecountF, f := openf("pagecount.txt", 0)
	defer f()

	if len(shaMap) == 0 {
		fmt.Fprintf(resf, "page\tindex\tSHA\tNAME\tPath\tURL\n")
	}

	for {
		if *startPage > 3000 {
			break
		}
		result, err := searchProtoOpts(ctx, *startPage, 100, false)
		if err != nil {
			fmt.Printf("3 error %v\n", err)
			break
		}
		for j, r := range result.CodeResults {
			if count, ex := shaMap[*r.SHA]; ex {
				shaMap[*r.SHA] = count + 1
				continue
			}
			shaMap[*r.SHA] = 1
			if r.TextMatches != nil {
				_, err = fmt.Fprintf(tmf, "%d\t%d\t%s\t%s\t%s\t%s\n%v\n------------------\n", *startPage, j, *r.SHA, *r.Name, *r.Path, *r.HTMLURL, r.TextMatches)
				if err != nil {
					log.Printf("%v", err)
				}
			}
			_, err = fmt.Fprintf(resf, "%d\t%d\t%s\t%s\t%s\t%s\n", *startPage, j, *r.SHA, *r.Name, *r.Path, *r.HTMLURL)
			if err != nil {
				log.Printf("%v", err)
			}
		}
		fmt.Print(".")
		resf.Flush()
		*startPage = *startPage + 1
	}
	fmt.Printf("\n")
	fmt.Fprintf(pagecountF, "page %d\n", *startPage)
	for k, v := range shaMap {
		fmt.Fprintf(shaCount, "%s\t%d\n", k, v)
	}
}
