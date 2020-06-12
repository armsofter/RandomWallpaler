package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/jasonlvhit/gocron"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gotk3/gotk3/gtk"
	"github.com/reujab/wallpaper"
)

var (
	// Command line flags
	outputFile     string
	updateHour     uint64
	source         string
	startedService string
)

const maxFetchingCount int = 10

// https://unsplash.it/3840/2160/?random
// https://source.unsplash.com/1920x1080/?bmw
func init() {
	flag.StringVar(&outputFile, "o", "", "output file for logs")
	flag.StringVar(&source, "s", "rand", "bing: gets photos from Bing.com (max 8 for day) \n rand: or gets photos unsplash.it 4k random wallpaper")

	flag.Usage = usage
}

func usage() {
	fmt.Fprintf(os.Stderr, "USAGE: %s [OPTIONS]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "OPTIONS:\n")
	flag.PrintDefaults()
}

func main() {

	gtk.Init(nil)

	b, err := gtk.BuilderNew()
	if err != nil {
		log.Fatal("error:", err)
	}

	err = b.AddFromFile("main.glade")
	if err != nil {
		log.Fatal("error:", err)
	}

	obj, err := b.GetObject("window_main")
	if err != nil {
		log.Fatal("error:", err)
	}

	win := obj.(*gtk.ApplicationWindow)
	win.Connect("destroy", func() {
		gtk.MainQuit()
	})

	win.ShowAll()

	obj, _ = b.GetObject("service_loader")
	service_loader := obj.(*gtk.Spinner)

	service_loader.Hide()

	obj, _ = b.GetObject("started_label")
	started_label := obj.(*gtk.Label)

	started_label.Hide()

	obj, _ = b.GetObject("cron_timer1")
	cron_timer1 := obj.(*gtk.Entry)

	obj, _ = b.GetObject("stop_service")
	stop_service := obj.(*gtk.Button)

	stop_service.Hide()

	obj, _ = b.GetObject("start_random_service")
	start_random_service := obj.(*gtk.Button)

	obj, _ = b.GetObject("start_bing_service")
	start_bing_service := obj.(*gtk.Button)

	stop_service.Connect("clicked", func() {
		gocron.Clear()
		start_random_service.Show()
		start_bing_service.Show()
		service_loader.Hide()
		stop_service.Hide()
		started_label.Hide()
	})

	// starts cron for Bing service
	start_bing_service.Connect("clicked", func() {
		s, _ := cron_timer1.GetText()
		time, err := parseTimer(s)
		if err == nil {
			updateHour = time
			startWallpaperService()
			go startService()
			stop_service.Show()
			started_label.Show()
			service_loader.Show()
			start_random_service.Hide()
			start_bing_service.Hide()
			startedService = "Bing"
			source = "bing"
		}

	})

	// starts cron for random service
	start_random_service.Connect("clicked", func() {
		s, _ := cron_timer1.GetText()
		time, err := parseTimer(s)
		if err == nil {
			updateHour = time
			startWallpaperService()
			go startService()
			stop_service.Show()
			started_label.Show()
			service_loader.Show()
			started_label.SetLabel(source + " Service is started...")
			start_random_service.Hide()
			start_bing_service.Hide()
			startedService = "Random"
			source = "rand"
		}

	})

	obj, _ = b.GetObject("start_rand")
	start_rand_button := obj.(*gtk.Button)

	start_rand_button.Connect("clicked", func() {
		source = "rand"
		startParsing()
	})

	obj, _ = b.GetObject("start_bing")
	start_bing_button := obj.(*gtk.Button)

	start_bing_button.Connect("clicked", func() {
		source = "bing"
		startParsing()
	})

	gtk.Main()
}

func startParsing() {
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

	log.SetPrefix("[RandomWallpaper] ")
	log.SetOutput(output)

	start()
}

func parseTimer(hour string) (uint64, error) {
	value, err := strconv.ParseUint(hour, 10, 64)
	if err == nil {
		return value, nil
	}
	return 0, errors.New("error on parsing timer");
}

func startWallpaperService() {
	// set first time on launch
	if source == "bing" {
		fmt.Println("Bing source selected")
		setBingWallpaper()
	} else if source == "rand" {
		fmt.Println("unsp selected")
		setUnspWallpaper()
	}
}

func start() {
	startWallpaperService()
}

func startService() {
	if err := gocron.Every(updateHour).Seconds().Do(startWallpaperService); err != nil {
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
	image := image{URL: "https://unsplash.it/1920/1080/?random", Copyright: "None"}
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
	url := "https://www.bing.com/HPImageArchive.aspx?format=js&idx=0&n=" + strconv.Itoa(maxFetchingCount)
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

	if len(root.Images) < 1 {
		return nil, errors.New("response does not contain an image")
	}

	randomIndex := getRandInRange(0, len(root.Images)-1)
	fmt.Println(randomIndex)
	fmt.Println(len(root.Images))
	image := root.Images[randomIndex]
	image.URL = "https://www.bing.com" + image.URL

	return &image, nil
}

type IntRange struct {
	min, max int
}

// get next random value within the interval including min and max
func (ir *IntRange) NextRandom(r *rand.Rand) int {
	return r.Intn(ir.max-ir.min+1) + ir.min
}

func getRandInRange(min int, max int) int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	ir := IntRange{min, max}
	return ir.NextRandom(r)
}
