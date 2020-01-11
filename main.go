package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
)

type (
	Promo struct {
		Title          string `json:"title"`
		URL            string `json:"url"`
		ImageURL       string `json:"image_url"`
		AreaPromo      string `json:"area"`
		Periode        string `json:"periode"`
		ImageLargetURL string `json:"image_large"`
	}
	Paging struct {
		ID int `json:"id"`
	}
)

const (
	mainURL = "https://www.bankmega.com/"
)

// GetPaging
func GetPaging(ch chan<- Paging, url string) {
	resPaging, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resPaging.Body.Close()

	if resPaging.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", resPaging.StatusCode, resPaging.Status)
	}
	doc, err := goquery.NewDocumentFromReader(resPaging.Body)
	if err != nil {
		log.Fatal(err)
	}

	pagenya := 0
	doc.Find(".tablepaging tr td").Each(func(i int, s *goquery.Selection) {
		page, _ := s.Find("a").Attr("page")
		pageInt, _ := strconv.Atoi(page)
		if pageInt != pagenya {
			pagenya = pageInt
			if pagenya >= i {
				ch <- Paging{ID: pagenya}
			}

		}
	})
	close(ch)
}

// getData
func getData(chs chan<- Promo, id int, wg *sync.WaitGroup, url string) {
	defer wg.Done()
	resPaging, err := http.Get(fmt.Sprintf("%v&page=%d", url, id))
	if err != nil {
		log.Fatal(err)
	}
	defer resPaging.Body.Close()

	if resPaging.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", resPaging.StatusCode, resPaging.Status)
	}
	doc, err := goquery.NewDocumentFromReader(resPaging.Body)
	if err != nil {
		log.Fatal(err)
	}

	doc.Find("#promolain li").Each(func(i int, s *goquery.Selection) {
		URL, _ := s.Find("a").Attr("href")
		title, _ := s.Find("img").Attr("title")
		imageURL, _ := s.Find("img").Attr("src")

		resDetail, err := http.Get(mainURL + URL)
		fmt.Printf("%d. %v\n", i, mainURL+URL)

		if err != nil {
			log.Fatal(err)
		}

		defer resDetail.Body.Close()

		if resDetail.StatusCode != 200 {
			log.Fatalf("status code error: %d %s", resDetail.StatusCode, resDetail.Status)
		}

		docDetail, err := goquery.NewDocumentFromReader(resDetail.Body)
		if err != nil {
			log.Fatal(err)
		}

		var area, periode, imageLarge string
		docDetail.Find("#contentpromolain2").Each(func(i int, s *goquery.Selection) {
			area = s.Find(".area b").Text()
			periode = s.Find(".periode b").Text()
			imageLarge, _ = s.Find(".keteranganinside img").Attr("src")
		})
		chs <- Promo{
			URL:            mainURL + URL,
			Title:          title,
			ImageURL:       mainURL + imageURL,
			AreaPromo:      area,
			Periode:        periode,
			ImageLargetURL: mainURL + imageLarge,
		}
	})

}

// Fetch Data
func Fetch(url string) []Promo {

	fmt.Printf("=> %v\n", url)

	var wg sync.WaitGroup
	cat1 := make(chan Paging, 0)
	catData1 := make(chan Promo, 0)

	go func() {
		GetPaging(cat1, url)
	}()

	for x := range cat1 {
		wg.Add(1)
		go getData(catData1, x.ID, &wg, url)
	}

	go func() {
		wg.Wait()
		close(catData1)
	}()

	promos := make([]Promo, 0)

	i := 0
	for xs := range catData1 {
		i++
		promos = append(promos, Promo{
			xs.Title,
			xs.URL,
			xs.ImageURL,
			xs.AreaPromo,
			xs.Periode,
			xs.ImageLargetURL,
		})
	}
	return promos

}

func main() {
	// set proccessor running
	runtime.GOMAXPROCS(4)
	os.Remove("solution.json")
	// list category
	sub := map[string]int{
		"travel and Entertaiment":   1,
		"lifeStyle and welness":     2,
		"f And b":                   3,
		"Gadget and Electronics":    4,
		"daily Need and appliances": 5,
		"etc":                       6,
	}
	result := []map[string]interface{}{}
	fmt.Printf("Mulai...\n")
	jobs := make(chan bool, 1)
	// start crawel
	for x, y := range sub {
		datas := Fetch(fmt.Sprintf("%v=%d",
			"https://www.bankmega.com/promolainnya.php?product=0&subcat", y))
		ms := map[string]interface{}{
			x: datas,
		}
		result = append(result, ms)
	}
	close(jobs)
	<-jobs

	fmt.Printf("Selesai..\n")

	res, err := json.MarshalIndent(result, "", "\t")

	if err != nil {
		log.Fatal(err)
	}

	if err = ioutil.WriteFile("solution.json", res, 0644); err != nil {
		log.Fatal(err)
	}
}
