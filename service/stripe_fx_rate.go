package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	"github.com/shopspring/decimal"
	"golang.org/x/net/html"
)

const (
	stripeFXRefreshInterval       = 15 * time.Minute
	stripeFXRefreshHourBeijing    = 11
	stripeFXHTTPTimeout           = 10 * time.Second
	stripeFXMaximumResponseBytes  = 512 << 10
	stripeFXMaximumDailyChangePct = int64(3)
)

type BOCStripeFXQuote struct {
	RateE4      int64
	PublishedAt time.Time
}

type StripeFXRefreshResult struct {
	PricingDate       string `json:"pricing_date"`
	RateE4            int64  `json:"rate_e4"`
	SourcePublishedAt int64  `json:"source_published_at"`
}

var (
	stripeFXNow          = time.Now
	stripeFXFetchBOCRate = FetchBOCStripeFXRate
	stripeFXHTTPClient   = newBOCStripeFXHTTPClient()
)

func newBOCStripeFXHTTPClient() *http.Client {
	return &http.Client{
		Timeout: stripeFXHTTPTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("Bank of China FX redirect limit exceeded")
			}
			if req.URL.Scheme != "https" || req.URL.Hostname() != "www.boc.cn" {
				return errors.New("Bank of China FX redirect left the trusted HTTPS host")
			}
			return nil
		},
	}
}

// FetchBOCStripeFXRate reads the official Bank of China price table from a
// fixed HTTPS origin. It extracts only the USD spot-selling quote and source
// publication time; no customer-controlled URL is accepted.
func FetchBOCStripeFXRate(ctx context.Context) (*BOCStripeFXQuote, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, model.StripeFXBOCSourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create Bank of China FX request: %w", err)
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("User-Agent", "UniRouters-FX/1.0")

	resp, err := stripeFXHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch Bank of China FX rate: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Bank of China FX response status is %d", resp.StatusCode)
	}
	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || (mediaType != "text/html" && mediaType != "application/xhtml+xml") {
		return nil, errors.New("Bank of China FX response is not HTML")
	}

	limitedBody := io.LimitReader(resp.Body, stripeFXMaximumResponseBytes+1)
	body, err := io.ReadAll(limitedBody)
	if err != nil {
		return nil, fmt.Errorf("read Bank of China FX response: %w", err)
	}
	if len(body) > stripeFXMaximumResponseBytes {
		return nil, errors.New("Bank of China FX response exceeds the size limit")
	}
	quote, err := parseBOCUSDSpotSellingHTML(body)
	if err != nil {
		return nil, fmt.Errorf("parse Bank of China FX response: %w", err)
	}
	return quote, nil
}

func parseBOCUSDSpotSellingHTML(body []byte) (*BOCStripeFXQuote, error) {
	document, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var matchingRows []*html.Node
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "tr" {
			for _, attribute := range node.Attr {
				if attribute.Key == "data-currency" && strings.TrimSpace(attribute.Val) == "美元" {
					matchingRows = append(matchingRows, node)
					break
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(document)
	if len(matchingRows) != 1 {
		return nil, fmt.Errorf("expected one USD row, found %d", len(matchingRows))
	}

	columns := directHTMLCells(matchingRows[0], "td")
	headers, err := tableHeadersForBOCRow(matchingRows[0])
	if err != nil {
		return nil, err
	}
	if len(columns) != len(headers) {
		return nil, errors.New("Bank of China USD row has an unexpected shape")
	}
	currencyIndex, err := uniqueBOCHeaderIndex(headers, "货币名称")
	if err != nil {
		return nil, err
	}
	spotSellingIndex, err := uniqueBOCHeaderIndex(headers, "现汇卖出价")
	if err != nil {
		return nil, err
	}
	publishedAtIndex, err := uniqueBOCHeaderIndex(headers, "发布日期")
	if err != nil {
		return nil, err
	}
	if columns[currencyIndex] != "美元" {
		return nil, errors.New("Bank of China USD row currency does not match its header")
	}

	quotedPerHundred, err := decimal.NewFromString(columns[spotSellingIndex])
	if err != nil || quotedPerHundred.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("Bank of China USD spot-selling rate is invalid")
	}
	scaled := quotedPerHundred.Mul(decimal.NewFromInt(100))
	if !scaled.Equal(scaled.Truncate(0)) {
		return nil, errors.New("Bank of China USD spot-selling rate has unsupported precision")
	}
	rateE4 := scaled.IntPart()
	if rateE4 < 50_000 || rateE4 > 100_000 {
		return nil, errors.New("Bank of China USD spot-selling rate is out of range")
	}

	publishedAt, err := time.ParseInLocation(
		"2006/01/02 15:04:05",
		columns[publishedAtIndex],
		model.StripeFXBeijingLocation(),
	)
	if err != nil {
		return nil, errors.New("Bank of China USD publication time is invalid")
	}
	return &BOCStripeFXQuote{RateE4: rateE4, PublishedAt: publishedAt}, nil
}

func directHTMLCells(row *html.Node, element string) []string {
	cells := make([]string, 0, 8)
	for child := row.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == element {
			cells = append(cells, normalizedHTMLText(child))
		}
	}
	return cells
}

func tableHeadersForBOCRow(row *html.Node) ([]string, error) {
	table := row.Parent
	for table != nil && (table.Type != html.ElementNode || table.Data != "table") {
		table = table.Parent
	}
	if table == nil {
		return nil, errors.New("Bank of China USD row has no containing table")
	}

	var headerRows [][]string
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "tr" {
			headers := directHTMLCells(node, "th")
			if len(headers) > 0 {
				headerRows = append(headerRows, headers)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(table)
	if len(headerRows) != 1 {
		return nil, fmt.Errorf("expected one Bank of China header row, found %d", len(headerRows))
	}
	return headerRows[0], nil
}

func uniqueBOCHeaderIndex(headers []string, expected string) (int, error) {
	index := -1
	for current, header := range headers {
		if header != expected {
			continue
		}
		if index >= 0 {
			return 0, fmt.Errorf("Bank of China header %q is duplicated", expected)
		}
		index = current
	}
	if index < 0 {
		return 0, fmt.Errorf("Bank of China header %q is missing", expected)
	}
	return index, nil
}

func normalizedHTMLText(node *html.Node) string {
	var builder strings.Builder
	var appendText func(*html.Node)
	appendText = func(current *html.Node) {
		if current.Type == html.TextNode {
			builder.WriteString(current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			appendText(child)
		}
	}
	appendText(node)
	return strings.Join(strings.Fields(builder.String()), " ")
}

// RefreshStripeFXDailyRate fetches and immutably selects the BOC rate for the
// Beijing pricing date containing now. Repeated calls return the existing row
// without another network request.
func RefreshStripeFXDailyRate(ctx context.Context, now time.Time) (*model.StripeFXDailyRate, error) {
	if now.IsZero() {
		return nil, model.ErrStripeFXRateInvalid
	}
	pricingDate := model.StripeFXPricingDate(now)
	existing, err := model.FindStripeFXDailyRateByPricingDate(pricingDate)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, model.ErrStripeFXRateUnavailable) {
		return nil, err
	}

	quote, err := stripeFXFetchBOCRate(ctx)
	if err != nil {
		return nil, err
	}
	if quote == nil || quote.RateE4 <= 0 || quote.PublishedAt.IsZero() {
		return nil, model.ErrStripeFXRateInvalid
	}
	if model.StripeFXPricingDate(quote.PublishedAt) != pricingDate {
		return nil, model.ErrStripeFXRateUnavailable
	}

	previous, previousErr := model.FindEffectiveStripeFXDailyRate(now)
	if previousErr != nil && !errors.Is(previousErr, model.ErrStripeFXRateUnavailable) {
		return nil, previousErr
	}
	if previous != nil {
		difference := quote.RateE4 - previous.RateE4
		if difference < 0 {
			difference = -difference
		}
		if difference*100 > previous.RateE4*stripeFXMaximumDailyChangePct {
			return nil, errors.New("Bank of China FX rate changed by more than 3 percent")
		}
	}

	rate := &model.StripeFXDailyRate{
		PricingDate:       pricingDate,
		Source:            model.StripeFXSourceBOCSpotSelling,
		BaseCurrency:      model.StripeFXBaseCurrencyUSD,
		QuoteCurrency:     model.StripeFXQuoteCurrencyCNY,
		RateE4:            quote.RateE4,
		SourcePublishedAt: quote.PublishedAt.Unix(),
		FetchedAt:         now.Unix(),
		SourceURL:         model.StripeFXBOCSourceURL,
	}
	return model.CreateStripeFXDailyRate(rate)
}

type stripeFXRefreshHandler struct{}

func (stripeFXRefreshHandler) Type() string { return model.SystemTaskTypeStripeFXRefresh }

func (stripeFXRefreshHandler) Enabled() bool {
	if setting.GetStripeProductId() == "" {
		return false
	}
	now := stripeFXNow()
	beijingNow := now.In(model.StripeFXBeijingLocation())
	if beijingNow.Hour() < stripeFXRefreshHourBeijing {
		return false
	}
	_, err := model.FindStripeFXDailyRateByPricingDate(model.StripeFXPricingDate(now))
	return errors.Is(err, model.ErrStripeFXRateUnavailable)
}

func (stripeFXRefreshHandler) Interval() time.Duration { return stripeFXRefreshInterval }

func (stripeFXRefreshHandler) NewPayload() any { return nil }

func (stripeFXRefreshHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	rate, err := RefreshStripeFXDailyRate(ctx, stripeFXNow())
	if err != nil {
		failSystemTask(task, runnerID, err)
		return
	}
	result := StripeFXRefreshResult{
		PricingDate:       rate.PricingDate,
		RateE4:            rate.RateE4,
		SourcePublishedAt: rate.SourcePublishedAt,
	}
	if err := model.FinishSystemTask(
		task.TaskID,
		runnerID,
		model.SystemTaskStatusSucceeded,
		result,
		"",
	); err != nil {
		logSystemTaskLockError(ctx, task, err)
	}
}

func init() {
	RegisterSystemTaskHandler(stripeFXRefreshHandler{})
}
