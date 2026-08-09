package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const bocUSDHTMLFixture = `<!doctype html><html><body><table><thead><tr>
<th>货币名称</th><th>现汇买入价</th><th>现钞买入价</th><th>现汇卖出价</th><th>现钞卖出价</th><th>中行折算价</th><th>发布日期</th><th>发布时间</th>
</tr></thead><tbody>
<tr data-currency='欧元'><td>欧元</td><td>780.00</td><td>780.00</td><td>790.00</td><td>790.00</td><td>785.00</td><td class="pjrq">2026/08/09 10:30:00</td><td>10:30:00</td></tr>
<tr data-currency='美元'><td><span>美元</span></td><td>673.72</td><td>673.72</td><td>676.55</td><td>676.55</td><td>679.04</td><td class="pjrq">2026/08/09 10:30:00</td><td>10:30:00</td></tr>
</tbody></table></body></html>`

type stripeFXRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn stripeFXRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func stripeFXServiceTestTime(day int, hour int, minute int) time.Time {
	return time.Date(2026, time.August, day, hour, minute, 0, 0, model.StripeFXBeijingLocation())
}

func stripeFXServiceTestModelRate(now time.Time, rateE4 int64) *model.StripeFXDailyRate {
	return &model.StripeFXDailyRate{
		PricingDate:       model.StripeFXPricingDate(now),
		Source:            model.StripeFXSourceBOCSpotSelling,
		BaseCurrency:      model.StripeFXBaseCurrencyUSD,
		QuoteCurrency:     model.StripeFXQuoteCurrencyCNY,
		RateE4:            rateE4,
		SourcePublishedAt: now.Add(-90 * time.Minute).Unix(),
		FetchedAt:         now.Unix(),
		SourceURL:         model.StripeFXBOCSourceURL,
	}
}

func TestParseBOCUSDSpotSellingHTMLPreservesFourDecimalRate(t *testing.T) {
	quote, err := parseBOCUSDSpotSellingHTML([]byte(bocUSDHTMLFixture))
	require.NoError(t, err)
	assert.EqualValues(t, 67_655, quote.RateE4)
	assert.Equal(
		t,
		stripeFXServiceTestTime(9, 10, 30).Unix(),
		quote.PublishedAt.Unix(),
	)
}

func TestParseBOCUSDSpotSellingHTMLRejectsAmbiguousOrMalformedRows(t *testing.T) {
	for name, body := range map[string]string{
		"missing USD":                      `<table><tr data-currency='欧元'><td>欧元</td></tr></table>`,
		"duplicate USD":                    bocUSDHTMLFixture + `<table><tr data-currency='美元'><td>美元</td></tr></table>`,
		"fraction beyond source precision": strings.Replace(bocUSDHTMLFixture, "676.55", "676.551", 1),
		"out of range":                     strings.Replace(bocUSDHTMLFixture, "676.55", "1200.00", 1),
		"bad publication time":             strings.ReplaceAll(bocUSDHTMLFixture, "2026/08/09 10:30:00", "not-a-time"),
		"missing spot selling header":      strings.Replace(bocUSDHTMLFixture, "现汇卖出价", "其他价格", 1),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseBOCUSDSpotSellingHTML([]byte(body))
			require.Error(t, err)
		})
	}
}

func TestParseBOCUSDSpotSellingHTMLFollowsVerifiedHeaderOrder(t *testing.T) {
	reordered := `<!doctype html><html><body><table><thead><tr>
<th>发布日期</th><th>现汇卖出价</th><th>货币名称</th>
</tr></thead><tbody><tr data-currency='美元'>
<td>2026/08/09 10:30:00</td><td>676.55</td><td>美元</td>
</tr></tbody></table></body></html>`

	quote, err := parseBOCUSDSpotSellingHTML([]byte(reordered))
	require.NoError(t, err)
	assert.EqualValues(t, 67_655, quote.RateE4)
}

func TestFetchBOCStripeFXRateUsesOnlyFixedHTTPSOrigin(t *testing.T) {
	originalClient := stripeFXHTTPClient
	t.Cleanup(func() { stripeFXHTTPClient = originalClient })
	stripeFXHTTPClient = &http.Client{
		Transport: stripeFXRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, model.StripeFXBOCSourceURL, req.URL.String())
			assert.Equal(t, "UniRouters-FX/1.0", req.Header.Get("User-Agent"))
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
				Body:       io.NopCloser(strings.NewReader(bocUSDHTMLFixture)),
				Request:    req,
			}, nil
		}),
		Timeout: time.Second,
	}

	quote, err := FetchBOCStripeFXRate(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, 67_655, quote.RateE4)
}

func TestFetchBOCStripeFXRateLive(t *testing.T) {
	if os.Getenv("STRIPE_FX_LIVE_TEST") != "1" {
		t.Skip("set STRIPE_FX_LIVE_TEST=1 to query the official BOC page")
	}
	quote, err := FetchBOCStripeFXRate(context.Background())
	require.NoError(t, err)
	require.NotNil(t, quote)
	assert.GreaterOrEqual(t, quote.RateE4, int64(50_000))
	assert.LessOrEqual(t, quote.RateE4, int64(100_000))
	assert.Equal(
		t,
		model.StripeFXPricingDate(time.Now()),
		model.StripeFXPricingDate(quote.PublishedAt),
	)
}

func TestFetchBOCStripeFXRateRejectsOversizedBody(t *testing.T) {
	originalClient := stripeFXHTTPClient
	t.Cleanup(func() { stripeFXHTTPClient = originalClient })
	stripeFXHTTPClient = &http.Client{Transport: stripeFXRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/html"}},
			Body:       io.NopCloser(strings.NewReader(strings.Repeat("x", stripeFXMaximumResponseBytes+1))),
			Request:    req,
		}, nil
	})}

	_, err := FetchBOCStripeFXRate(context.Background())
	require.ErrorContains(t, err, "size limit")
}

func TestBOCStripeFXClientRejectsUntrustedRedirects(t *testing.T) {
	client := newBOCStripeFXHTTPClient()
	evilURL, err := url.Parse("http://example.com/rate")
	require.NoError(t, err)
	err = client.CheckRedirect(&http.Request{URL: evilURL}, nil)
	require.ErrorContains(t, err, "trusted HTTPS host")

	trustedURL, err := url.Parse(model.StripeFXBOCSourceURL + "index.html")
	require.NoError(t, err)
	require.NoError(t, client.CheckRedirect(&http.Request{URL: trustedURL}, nil))
}

func TestRefreshStripeFXDailyRatePersistsOnceAndAvoidsSecondFetch(t *testing.T) {
	truncate(t)
	originalFetch := stripeFXFetchBOCRate
	t.Cleanup(func() { stripeFXFetchBOCRate = originalFetch })
	now := stripeFXServiceTestTime(9, 13, 45)
	fetchCount := 0
	stripeFXFetchBOCRate = func(context.Context) (*BOCStripeFXQuote, error) {
		fetchCount++
		return &BOCStripeFXQuote{
			RateE4:      67_655,
			PublishedAt: stripeFXServiceTestTime(9, 10, 30),
		}, nil
	}

	first, err := RefreshStripeFXDailyRate(context.Background(), now)
	require.NoError(t, err)
	second, err := RefreshStripeFXDailyRate(context.Background(), now.Add(10*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, first.Id, second.Id)
	assert.Equal(t, 1, fetchCount)
	assert.Equal(t, "6.7655", first.RateString())
}

func TestRefreshStripeFXDailyRateRejectsLargeDailyDeviation(t *testing.T) {
	truncate(t)
	originalFetch := stripeFXFetchBOCRate
	t.Cleanup(func() { stripeFXFetchBOCRate = originalFetch })
	previousNow := stripeFXServiceTestTime(8, 12, 0)
	_, err := model.CreateStripeFXDailyRate(stripeFXServiceTestModelRate(previousNow, 67_600))
	require.NoError(t, err)
	currentNow := stripeFXServiceTestTime(9, 13, 45)
	stripeFXFetchBOCRate = func(context.Context) (*BOCStripeFXQuote, error) {
		return &BOCStripeFXQuote{
			RateE4:      70_000,
			PublishedAt: stripeFXServiceTestTime(9, 10, 30),
		}, nil
	}

	_, err = RefreshStripeFXDailyRate(context.Background(), currentNow)
	require.ErrorContains(t, err, "more than 3 percent")
	_, err = model.FindStripeFXDailyRateByPricingDate(model.StripeFXPricingDate(currentNow))
	require.ErrorIs(t, err, model.ErrStripeFXRateUnavailable)
}

func TestRefreshStripeFXDailyRateNeverLabelsPreviousDateAsToday(t *testing.T) {
	truncate(t)
	originalFetch := stripeFXFetchBOCRate
	t.Cleanup(func() { stripeFXFetchBOCRate = originalFetch })
	currentNow := stripeFXServiceTestTime(9, 13, 45)
	stripeFXFetchBOCRate = func(context.Context) (*BOCStripeFXQuote, error) {
		return &BOCStripeFXQuote{
			RateE4:      67_655,
			PublishedAt: stripeFXServiceTestTime(8, 10, 30),
		}, nil
	}

	_, err := RefreshStripeFXDailyRate(context.Background(), currentNow)
	require.ErrorIs(t, err, model.ErrStripeFXRateUnavailable)
	_, err = model.FindStripeFXDailyRateByPricingDate(model.StripeFXPricingDate(currentNow))
	require.ErrorIs(t, err, model.ErrStripeFXRateUnavailable)
}

func TestStripeFXRefreshHandlerRunsOnlyAfterElevenUntilDailyRateExists(t *testing.T) {
	truncate(t)
	t.Setenv("STRIPE_PRODUCT_ID", "prod_daily_fx")
	originalNow := stripeFXNow
	t.Cleanup(func() { stripeFXNow = originalNow })
	handler := stripeFXRefreshHandler{}

	stripeFXNow = func() time.Time { return stripeFXServiceTestTime(9, 10, 59) }
	assert.False(t, handler.Enabled())
	stripeFXNow = func() time.Time { return stripeFXServiceTestTime(9, 11, 0) }
	assert.True(t, handler.Enabled())
	assert.Equal(t, 15*time.Minute, handler.Interval())

	_, err := model.CreateStripeFXDailyRate(stripeFXServiceTestModelRate(stripeFXServiceTestTime(9, 11, 5), 67_655))
	require.NoError(t, err)
	assert.False(t, handler.Enabled())
}

func TestStripeFXRefreshHandlerCompletesLeasedSystemTask(t *testing.T) {
	truncate(t)
	originalNow := stripeFXNow
	originalFetch := stripeFXFetchBOCRate
	t.Cleanup(func() {
		stripeFXNow = originalNow
		stripeFXFetchBOCRate = originalFetch
	})
	now := stripeFXServiceTestTime(9, 13, 45)
	stripeFXNow = func() time.Time { return now }
	stripeFXFetchBOCRate = func(context.Context) (*BOCStripeFXQuote, error) {
		return &BOCStripeFXQuote{RateE4: 67_655, PublishedAt: stripeFXServiceTestTime(9, 10, 30)}, nil
	}
	task, err := model.CreateSystemTask(model.SystemTaskTypeStripeFXRefresh, nil, nil)
	require.NoError(t, err)
	claimed, ok, err := model.ClaimSystemTask(task.ID, task.Type, "fx-runner", common.GetTimestamp()+60)
	require.NoError(t, err)
	require.True(t, ok)

	stripeFXRefreshHandler{}.Run(context.Background(), claimed, "fx-runner")
	finished, err := model.GetSystemTaskByTaskID(task.TaskID)
	require.NoError(t, err)
	assert.Equal(t, model.SystemTaskStatusSucceeded, finished.Status)
	rate, err := model.FindStripeFXDailyRateByPricingDate(model.StripeFXPricingDate(now))
	require.NoError(t, err)
	assert.EqualValues(t, 67_655, rate.RateE4)
}

func TestStripeFXRefreshHandlerRetriesOnlyAfterFailedRunInterval(t *testing.T) {
	truncate(t)
	t.Setenv("STRIPE_PRODUCT_ID", "prod_daily_fx")
	originalNow := stripeFXNow
	originalFetch := stripeFXFetchBOCRate
	t.Cleanup(func() {
		stripeFXNow = originalNow
		stripeFXFetchBOCRate = originalFetch
	})
	now := stripeFXServiceTestTime(9, 13, 45)
	stripeFXNow = func() time.Time { return now }
	stripeFXFetchBOCRate = func(context.Context) (*BOCStripeFXQuote, error) {
		return nil, errors.New("temporary BOC failure")
	}
	handler := stripeFXRefreshHandler{}
	withSystemTaskRegistry(t, handler)
	task, err := model.CreateSystemTask(handler.Type(), nil, nil)
	require.NoError(t, err)
	claimed, ok, err := model.ClaimSystemTask(task.ID, task.Type, "fx-retry-runner", common.GetTimestamp()+60)
	require.NoError(t, err)
	require.True(t, ok)
	handler.Run(context.Background(), claimed, "fx-retry-runner")

	failed, err := model.GetSystemTaskByTaskID(task.TaskID)
	require.NoError(t, err)
	require.Equal(t, model.SystemTaskStatusFailed, failed.Status)
	runSystemTaskScheduler()
	assert.EqualValues(t, 1, countSystemTasks(t, handler.Type()))

	require.NoError(t, model.DB.Model(&model.SystemTask{}).
		Where("task_id = ?", task.TaskID).
		Update("updated_at", common.GetTimestamp()-int64((16*time.Minute).Seconds())).Error)
	runSystemTaskScheduler()
	runSystemTaskScheduler()
	assert.EqualValues(t, 2, countSystemTasks(t, handler.Type()))
}

func TestStripeFXRefreshHandlerIsRegistered(t *testing.T) {
	found := false
	for _, handler := range registeredSystemTaskHandlers() {
		if handler.Type() == model.SystemTaskTypeStripeFXRefresh {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestRefreshStripeFXDailyRatePropagatesFetchFailureWithoutPersisting(t *testing.T) {
	truncate(t)
	originalFetch := stripeFXFetchBOCRate
	t.Cleanup(func() { stripeFXFetchBOCRate = originalFetch })
	stripeFXFetchBOCRate = func(context.Context) (*BOCStripeFXQuote, error) {
		return nil, errors.New("upstream unavailable")
	}
	now := stripeFXServiceTestTime(9, 13, 45)

	_, err := RefreshStripeFXDailyRate(context.Background(), now)
	require.ErrorContains(t, err, "upstream unavailable")
	_, err = model.FindStripeFXDailyRateByPricingDate(model.StripeFXPricingDate(now))
	require.ErrorIs(t, err, model.ErrStripeFXRateUnavailable)
}
