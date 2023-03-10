package vctrl

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	config "github.com/moqsien/gvc/pkgs/confs"
	"github.com/moqsien/gvc/pkgs/downloader"
	"github.com/moqsien/gvc/pkgs/utils"
)

var Platform map[string]string = map[string]string{
	"darwin":  "mac",
	"windows": "windows",
	"linux":   "linux",
	"amd64":   "x64",
	"arm64":   "arm64",
}

type JavaPackage struct {
	Url      string
	FileName string
	OS       string
	Arch     string
	Size     string
	Checksum string
}

type JavaVersion struct {
	c        *colly.Collector
	d        *downloader.Downloader
	Versions map[string][]*JavaPackage
	Doc      *goquery.Document
	Conf     *config.GVConfig
}

func NewJavaVersion() (jv *JavaVersion) {
	jv = &JavaVersion{
		Versions: make(map[string][]*JavaPackage, 50),
		Conf:     config.New(),
		c:        colly.NewCollector(),
		d:        &downloader.Downloader{},
	}
	jv.initeDirs()
	return
}

func (that *JavaVersion) initeDirs() {
	if ok, _ := utils.PathIsExist(config.DefaultJavaRoot); !ok {
		if err := os.MkdirAll(config.DefaultJavaRoot, os.ModePerm); err != nil {
			fmt.Println("[mkdir Failed] ", err)
		}
	}
	if ok, _ := utils.PathIsExist(config.JavaTarFilesPath); !ok {
		if err := os.MkdirAll(config.JavaTarFilesPath, os.ModePerm); err != nil {
			fmt.Println("[mkdir Failed] ", err)
		}
	}
	if ok, _ := utils.PathIsExist(config.JavaUnTarFilesPath); !ok {
		if err := os.MkdirAll(config.JavaUnTarFilesPath, os.ModePerm); err != nil {
			fmt.Println("[mkdir Failed] ", err)
		}
	}
}

func (that *JavaVersion) getDoc() {
	if that.Conf.Java.CompilerUrl != "" {
		that.c.OnResponse(func(r *colly.Response) {
			// fmt.Println(string(r.Body))
			that.Doc, _ = goquery.NewDocumentFromReader(bytes.NewBuffer(r.Body))
		})
		if _, err := url.Parse(that.Conf.Java.CompilerUrl); err != nil {
			panic(err)
		}
		that.c.Visit(that.Conf.Java.CompilerUrl)
	}
}

func (that *JavaVersion) getSha(sUrl string) (res string) {
	if _, err := url.Parse(sUrl); err != nil {
		panic(err)
	}
	c := colly.NewCollector()
	c.OnResponse(func(r *colly.Response) {
		res = string(r.Body)
	})
	c.Visit(sUrl)
	return
}

func (that *JavaVersion) getVersions() {
	if that.Doc == nil {
		that.getDoc()
	}
	// time.Sleep(time.Second * 5)
	that.Doc.Find("ul.rw-inpagetabs").First().Find("li").Each(func(i int, s *goquery.Selection) {
		v, _ := s.Find("a").Attr("href")
		sList := strings.Split(v, "java")
		vn := sList[len(sList)-1]
		that.Doc.Find(fmt.Sprintf("div#java%s", vn)).After("nav").Find("table").Find("tbody").Find("tr").Each(func(i int, s *goquery.Selection) {
			if i == 0 {
				return
			}
			tArchive := strings.ToLower(s.Find("td").Eq(0).Text())
			tSize := s.Find("td").Eq(1).Text()
			tUrl, _ := s.Find("td").Eq(2).Find("a").Eq(0).Attr("href")
			tSha, _ := s.Find("td").Eq(2).Find("a").Eq(1).Attr("href")
			if strings.Contains(tArchive, Platform[runtime.GOARCH]) && strings.Contains(tArchive, "archive") {
				if !strings.Contains(tUrl, Platform[runtime.GOOS]) {
					return
				}
				p := &JavaPackage{}
				p.Arch = runtime.GOARCH
				p.OS = runtime.GOOS
				p.Size = tSize
				p.Url = tUrl
				fName := strings.Split(tUrl, "/")
				p.FileName = fName[len(fName)-1]
				p.Checksum = that.getSha(tSha)
				that.Versions[vn] = append(that.Versions[vn], p)
			}
		})
	})
}

func (that *JavaVersion) ShowVersions() {
	that.getVersions()
	vList := []string{}
	for k := range that.Versions {
		vList = append(vList, fmt.Sprintf("java%s", k))
	}
	fmt.Println(strings.Join(vList, "  "))
}

func (that *JavaVersion) Download(version string) (r string) {
	vList := strings.Split(version, "java")
	v := vList[len(vList)-1]
	that.getVersions()
	if pList, ok := that.Versions[v]; ok {
		p := pList[0]
		that.d.Url = p.Url
		that.d.Timeout = 300 * time.Minute
		fpath := filepath.Join(config.JavaTarFilesPath, p.FileName)
		if size := that.d.GetFile(fpath, os.O_CREATE|os.O_WRONLY, 0644); size > 0 {
			if ok := utils.CheckFile(fpath, "sha256", p.Checksum); ok {
				return fpath
			} else {
				os.RemoveAll(fpath)
			}
		}
	}
	return
}
