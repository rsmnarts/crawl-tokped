package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type data struct {
	NameOfProduct string `json:"name_of_product"`
	Desc          string `json:"desc"`
	ImageLink     string `json:"image_link"`
	Price         string `json:"price"`
	Rating        string `json:"rating"`
	MerchantName  string `json:"merchant_name"`
}

const (
	TOTAL_DATA   = 100
	QUERY_SEARCH = "handphone"
)

func main() {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.NoDefaultBrowserCheck,
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3830.0 Safari/537.36"),
		chromedp.Flag("headless", false),
		chromedp.NoFirstRun,
		chromedp.Flag("ignore-certificate-errors", true),
		// chromedp.Flag("start-fullscreen", true),
		chromedp.WindowSize(2560, 1600),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	contextOpts := []chromedp.ContextOption{
		chromedp.WithLogf(log.Printf),
		chromedp.WithDebugf(log.Printf),
		chromedp.WithErrorf(log.Printf),
	}

	taskCtx, cancel := chromedp.NewContext(allocCtx, contextOpts...)
	defer cancel()
	page := 0

	var links []string
	for {
		links = append(links, getLinks(taskCtx, page+1, TOTAL_DATA, true)...)
		if len(links) >= TOTAL_DATA {
			break
		}
	}

	csvFile, err := os.Create("tokped.csv")
	if err != nil {
		log.Println("os create file error:", err)
	}
	defer csvFile.Close()

	var writer = csv.NewWriter(csvFile)

	writeHeader(writer)

	for k, n := range links {
		l, err := url.ParseRequestURI(n)
		if err != nil {
			log.Println("Not link:", n)
			continue
		}

		var result data
		if err := chromedp.Run(taskCtx,
			chromedp.Navigate(l.String()),
			chromedp.Text(`[data-testid="lblPDPDetailProductName"]`, &result.NameOfProduct, chromedp.ByQuery),
			chromedp.Text(`[data-testid="lblPDPDetailProductPrice"]`, &result.Price, chromedp.ByQuery),
			chromedp.AttributeValue(`[data-testid="PDPImageMain"] div div img`, "src", &result.ImageLink, nil, chromedp.ByQuery),
			chromedp.Text(`[data-testid="llbPDPFooterShopName"] > h2`, &result.MerchantName, chromedp.ByQuery),
			chromedp.Text(`[data-testid="lblPDPDetailProductRatingNumber"]`, &result.Rating, chromedp.ByQuery),
			chromedp.Text(`[data-testid="lblPDPDescriptionProduk"]`, &result.Desc, chromedp.ByQuery),
		); err != nil {
			log.Println("Error:", err)
		}
		result.Price = extractNumberStr(result.Price, "")
		result.Rating = extractNumberStr(result.Rating, ".")
		result.Desc = url.QueryEscape(result.Desc)

		writeData(writer, result)

		if k >= TOTAL_DATA-1 {
			break
		}
	}
	writer.Flush()
}

func writeHeader(writer *csv.Writer) {
	results := []string{
		"name",
		"desc",
		"image_link",
		"price",
		"rating",
		"merchant_name",
	}
	if err := writer.Write(results); err != nil {
		log.Println("Write CSV Error:", err)
	}
}

func writeData(writer *csv.Writer, result data) {
	results := []string{
		result.NameOfProduct,
		result.Desc,
		result.ImageLink,
		result.Price,
		result.Rating,
		result.MerchantName,
	}
	if err := writer.Write(results); err != nil {
		log.Println("Write CSV Error:", err)
	}
}

func getLinks(ctx context.Context, page, min int, firstPage bool) (out []string) {
	link := `https://www.tokopedia.com/search?st=product&q=` + QUERY_SEARCH + `&navsource=home`
	if page > 1 {
		link = fmt.Sprintf("%s&page%d", link, page)
	}

	var nodes []*cdp.Node
	if err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate(link),
		// chromedp.Click(`.unf-coachmark__next-button`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.EvaluateAsDevTools(`window.scrollTo(0,document.body.scrollHeight);`, nil),
		chromedp.WaitVisible(`[data-unf="pagination-item"]`, chromedp.ByQuery),
		chromedp.Nodes(`[data-testid="master-product-card"] div div div a`, &nodes, chromedp.ByQueryAll),
	); err != nil {
		log.Fatalln(err)
	}

	var linkBefore string
	for _, n := range nodes {
		link := n.AttributeValue("href")
		if linkBefore == link || strings.Contains(link, "/promo/") {
			continue
		}
		linkBefore = link
		out = append(out, link)
	}

	return
}
