package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jasonlvhit/gocron"
	"github.com/reujab/wallpaper"
)

var (
	// Command line flags
	outputFile string
	updateHour string
	killFlag   bool
	source     string
)

const maxFetchingCount int = 8
//https://unsplash.it/3840/2160/?random
func init() {
	flag.StringVar(&outputFile, "o", "", "output file for logs")
	flag.StringVar(&source, "s", "rand", "bing: gets photos from Bing.com (max 8 for day) \n rand: or gets photos unsplash.it 4k random wallpaper")
	flag.StringVar(&updateHour, "h", "10", "24-hour time when wallpaper is updated")
	flag.BoolVar(&killFlag, "k", false, "update wallpaper once and exit")

	flag.Usage = usage
}

func usage() {
	fmt.Fprintf(os.Stderr, "USAGE: %s [OPTIONS]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "OPTIONS:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()

	output := io.Writer(os.Stdout)

	if outputFile != "" {
		f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			log.Printf("[ERR] unable to open output file %q: %v", outputFile, err)
			os.Exit(1)
		}
		defer f.Close()
		output = f
	}

	log.SetPrefix("[RandomWallpaler] ")
	log.SetOutput(output)

	start()
}

func start() {
	// set first time on launch
	if source == "bing" {
		fmt.Println("Bing source selected")
		setBingWallpaper()
	} else if source == "rand" {
		fmt.Println("unsp selected")
		setUnspWallpaper()
	}

	if killFlag {
		log.Printf("[INF] kill flag provided, exiting")
		os.Exit(0)
	}

	// set again daily
	hour, _ := strconv.ParseUint(updateHour, 10, 64)
	if err := gocron.Every(hour).Seconds().Do(setBingWallpaper); err != nil {
		log.Printf("[ERR] failed to create daily update job at %q: %v", updateHour, err)
		os.Exit(1)
	}
	log.Printf("[INF] wallpaper will be updated again in every %s hour", updateHour)

	<-gocron.Start()
}

// setBingWallpaper sets the wallpaper to Bing's current image of the day,
// if it fails an error is logged.
func setBingWallpaper() {
	image, err := bingImageOfTheDay()
	if err != nil {
		log.Printf("[ERR] unable to retrieve Bing image of the day: %v", err)
		return
	}
	// append a dummy appendix for wallpaper module to recognize the filename
	image.URL = image.URL + "/fakefilename=2020.jpg"
	log.Printf("[INF] updating wallpaper, url: %q, copyright: %q", image.URL, image.Copyright)

	if err := wallpaper.SetFromURL(image.URL); err != nil {
		log.Printf("[ERR] unable to set wallpaper: %v", err)
	}
}


// setUnspWallpaper sets the wallpaper to https://unsplash.it random 4k photo
// if it fails an error is logged.
func setUnspWallpaper() {
	image := image{URL: "https://unsplash.it/3840/2160/?random", Copyright:"None"}
	image.URL = image.URL + "/fakefilename=2020.jpg"
	log.Printf("[INF] updating wallpaper, url: %q, copyright: %q", image.URL, image.Copyright)

	if err := wallpaper.SetFromURL(image.URL); err != nil {
		log.Printf("[ERR] unable to set wallpaper: %v", err)
	}
}


var client = http.Client{Timeout: 30 * time.Second}

type image struct {
	URL       string `json:"url"`
	Copyright string `json:"copyright"`
}

// bingImageOfTheDay returns Bing's current image of the day.
func bingImageOfTheDay() (*image, error) {
	url := "https://www.bing.com/HPImageArchive.aspx?format=js&idx=0&n="+ strconv.Itoa(maxFetchingCount)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http GET: %v", err)
	}
	defer resp.Body.Close()

	root := struct {
		Images []image `json:"images"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		return nil, fmt.Errorf("decode body: %v", err)
	}

	if len(root.Images) > maxFetchingCount {
		return nil, errors.New("response does not contain an image")
	}

	randomIndex := getRandInRange(0, maxFetchingCount)
	image := root.Images[randomIndex]
	image.URL = "https://www.bing.com" + image.URL

	return &image, nil
}

type IntRange struct {
	min, max int
}

// get next random value within the interval including min and max
func (ir *IntRange) NextRandom(r* rand.Rand) int {
	return r.Intn(ir.max - ir.min +1) + ir.min
}

func getRandInRange(min int, max int) int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	ir := IntRange{min, max}
	return ir.NextRandom(r)
}